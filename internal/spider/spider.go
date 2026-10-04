package spider

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"gomaccms/internal/collect"
	"gomaccms/internal/config"
	"gomaccms/internal/file"
	"gomaccms/internal/film"
	"gomaccms/internal/siteconfig"
	"gomaccms/internal/util"
	"log"
	"math"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

/*
	采集逻辑 v3

*/

var spiderCore = &JsonCollect{}

// ======================================================= 通用采集方法  =======================================================

// HandleCollectRefine 采集一个采集接口最近 h 小时更新的影片 (h < 0 为全部): 只采集已绑定本站分类的采集站分类
// (未单独绑定的子分类沿用其一级分类的绑定), 每部影片按入库重复规则与已有影片合并 (见 film.IngestVods);
// 采集进度见 Progresses
func HandleCollectRefine(id string, h int) error {
	s, err := prepareCollect(id, h, modeLabel(h), TriggerCron)
	if err != nil {
		return err
	}
	return runCollect(s, h, nil)
}

// StartCollect 登记采集任务后在后台执行; 登记是同步的, 同一采集接口重复点击时第二次直接返回错误
func StartCollect(id string, h int) error {
	s, err := prepareCollect(id, h, modeLabel(h), TriggerManual)
	if err != nil {
		return err
	}
	go func() {
		if err := runCollect(s, h, nil); err != nil {
			log.Printf("采集接口「%s」采集失败: %v", s.Name, err)
		}
	}()
	return nil
}

// ResumeCollect 从上次 (已中止 / 已中断 / 失败) 采集的中断处继续, 沿用上次的采集时长
func ResumeCollect(id string) error {
	from, last, err := lastResumePoint(id)
	if err != nil {
		return err
	}
	mode := last.Mode
	if !strings.HasPrefix(mode, "继续") {
		mode = "继续" + mode
	}
	s, err := prepareCollect(id, last.Hours, mode, TriggerResume)
	if err != nil {
		return err
	}
	go func() {
		if err := runCollect(s, last.Hours, from); err != nil {
			log.Printf("采集接口「%s」续采失败: %v", s.Name, err)
		}
	}()
	return nil
}

// RetryFailedPages 重采一笔采集记录中失败的页 (只请求该采集接口的这些页), 重试本身也会记一笔采集记录.
// 采集站按更新时间排序, 页码对应的影片会随新数据后移, 因此重试是按当前数据重新请求这些页码;
// 按时长采集时把时长加上距今的小时数, 让时间范围的起点不变
func RetryFailedPages(logId uint64) error {
	l, err := collect.Repo.FindCollectLog(logId)
	switch {
	case err != nil:
		return errors.New("采集记录不存在")
	case len(l.FailedList) == 0:
		return errors.New("该次采集没有失败的页")
	case l.Retried:
		return errors.New("该次采集的失败页已经重试过")
	}
	h := l.Hours
	if h > 0 {
		h += int(math.Ceil(time.Since(l.StartedAt).Hours()))
	}
	s, err := prepareCollect(l.SourceId, h, "重试失败页", TriggerRetry)
	if err != nil {
		return err
	}
	collect.Repo.MarkCollectLogRetried(l.Id)
	go func() {
		if err := runRetry(s, h, l.FailedList); err != nil {
			log.Printf("采集接口「%s」重试失败页失败: %v", s.Name, err)
		}
	}()
	return nil
}

// runRetry 逐页重采失败的页 (按采集站分类分组, 单线程), 每页之间检查是否被中止; 重试不能续采
func runRetry(s *collect.FilmSource, h int, pages []collect.FailedPage) (err error) {
	defer func() { finishProgress(s.Id, err) }()

	var types []int64
	byType := make(map[int64][]int)
	for _, fp := range pages {
		if _, ok := byType[fp.TypeId]; !ok {
			types = append(types, fp.TypeId)
		}
		byType[fp.TypeId] = append(byType[fp.TypeId], fp.Page)
	}
	names := classNames(s)
	trackProgress(s.Id, func(p *Progress) {
		p.TypeCount = len(types)
		p.logf("开始重试, 共 %d 页", len(pages))
	})
	r := util.RequestInfo{Uri: s.Uri, Params: s.Query()}
	if h > 0 {
		r.Params.Set("h", fmt.Sprint(h))
	}
	interval := collectInterval(s)
	for i, t := range types {
		if stopRequested(s.Id) {
			break
		}
		pgs := byType[t]
		trackProgress(s.Id, func(p *Progress) {
			p.startType(i+1, t, names[t], 1, len(pgs), 0)
			p.ResumeTypeId = 0
			p.logf("分类 %s: 重试第 %v 页", names[t], pgs)
		})
		r.Params.Set("t", fmt.Sprint(t))
		for _, pg := range pgs {
			if stopRequested(s.Id) {
				break
			}
			r.Params.Set("pg", fmt.Sprint(pg))
			collectFilmRefine(s, r)
			time.Sleep(time.Duration(interval) * time.Millisecond)
		}
	}
	ClearCache()
	return nil
}

// prepareCollect 校验采集接口并登记采集进度 (同一采集接口同时只能有一个采集任务)
func prepareCollect(id string, h int, mode, trigger string) (*collect.FilmSource, error) {
	s := collect.Repo.FindCollectSourceById(id)
	switch {
	case s == nil:
		return nil, errors.New("采集接口不存在")
	case !s.State:
		return nil, errors.New("采集接口未启用")
	case s.CollectType != collect.CollectVideo:
		return nil, errors.New("暂未开放此采集功能")
	case h == 0:
		return nil, errors.New("采集时长不能为 0")
	}
	if err := beginProgress(s, mode, trigger, h); err != nil {
		return nil, err
	}
	return s, nil
}

// runCollect 执行已登记的采集任务, 每页之间检查是否被中止; from 不为空时跳过之前的分类, 从其页码开始
func runCollect(s *collect.FilmSource, h int, from *resumePoint) (err error) {
	defer func() { finishProgress(s.Id, err) }()

	binds := effectiveBinds(s)
	types := collectTypes(binds)
	if len(types) == 0 {
		return fmt.Errorf("采集接口「%s」还没有绑定分类, 请先在「绑定」中把采集站分类绑定到本站分类", s.Name)
	}
	names := classNames(s)
	trackProgress(s.Id, func(p *Progress) {
		p.TypeCount = len(types)
		p.logf("开始%s, 共 %d 个已绑定的采集站分类", p.Mode, len(types))
		if from != nil {
			p.logf("从分类 %s 第 %d 页继续", names[from.TypeId], from.Page)
		}
	})
	r := util.RequestInfo{Uri: s.Uri, Params: s.Query()}
	if h > 0 {
		r.Params.Set("h", fmt.Sprint(h))
	}
	// 单次请求间隔 (ms): 采集接口未单独设置时使用后台「采集间隔」
	interval := collectInterval(s)
	for i, t := range types {
		if stopRequested(s.Id) {
			break
		}
		// 续采: 跳过已采集完的分类, 中断的分类从中断的页码开始
		startPage := 1
		if from != nil {
			if t < from.TypeId {
				continue
			}
			if t == from.TypeId {
				startPage = from.Page
			}
		}
		r.Params.Set("t", fmt.Sprint(t))
		pageCount, total, pageErr := spiderCore.GetPageInfo(r)
		if pageErr != nil {
			// 第二次获取分页页数依旧失败则跳过该分类
			pageCount, total, pageErr = spiderCore.GetPageInfo(r)
		}
		trackProgress(s.Id, func(p *Progress) {
			p.startType(i+1, t, names[t], startPage, pageCount, total)
			if pageErr != nil {
				p.logf("分类 %s 获取页数失败, 跳过: %v", names[t], pageErr)
				return
			}
			p.logf("分类 %s: 共 %d 页, %d 条", names[t], pageCount, total)
		})
		if pageErr != nil {
			continue
		}
		switch {
		case interval > 500:
			// 设置了较长的请求间隔: 单线程逐页采集
			for pg := startPage; pg <= pageCount && !stopRequested(s.Id); pg++ {
				r.Params.Set("pg", fmt.Sprint(pg))
				collectFilmRefine(s, r)
				time.Sleep(time.Duration(interval) * time.Millisecond)
			}
		case pageCount <= config.MAXGoroutine*5:
			for pg := startPage; pg <= pageCount && !stopRequested(s.Id); pg++ {
				r.Params.Set("pg", fmt.Sprint(pg))
				collectFilmRefine(s, r)
			}
		default:
			collectFilmMT(startPage, pageCount, s, r, collectFilmRefine)
		}
	}
	if s.SyncPictures() && !stopRequested(s.Id) {
		syncFilmPicture()
	}
	// 每次执行完都清理首页等接口数据缓存
	ClearCache()
	return nil
}

// collectFilmRefine 采集一页影片详情并入库, 失败的页记入本次采集 (采集记录中可重试)
func collectFilmRefine(s *collect.FilmSource, r util.RequestInfo) {
	list, err := spiderCore.GetFilmDetail(r)
	pg := r.Params.Get("pg")
	pgNum, _ := strconv.Atoi(pg)
	if err != nil || len(list) <= 0 {
		log.Printf("采集接口「%s」第 %s 页采集失败: %v", s.Name, pg, err)
		t, _ := strconv.ParseInt(r.Params.Get("t"), 10, 64)
		trackProgress(s.Id, func(p *Progress) {
			p.pageDone(pgNum)
			p.pageFailed(t, pgNum)
			p.logf("第 %s 页采集失败: %v", pg, err)
		})
		return
	}
	res := ingest(s, list)
	trackProgress(s.Id, func(p *Progress) {
		p.pageDone(pgNum)
		p.Added, p.Updated, p.Skipped = p.Added+res.Added, p.Updated+res.Updated, p.Skipped+res.Skipped
		for _, it := range res.Items {
			if it.Reason != "" {
				p.logf("第 %s 页  %s  %s (%s)", pg, it.Name, it.Action, it.Reason)
			} else {
				p.logf("第 %s 页  %s  %s", pg, it.Name, it.Action)
			}
		}
	})
}

// ingest 按分类绑定设置影片的本站分类后入库; 开启同步图片时登记新增影片的封面
func ingest(s *collect.FilmSource, list []film.MovieDetail) film.IngestResult {
	applyCategoryBind(effectiveBinds(s), list)
	res := film.IngestVods(s, siteconfig.Svc.GetSiteBasicConfig().Rule(), list)
	if s.SyncPictures() && len(res.NewPictures) > 0 {
		var pl []file.VirtualPicture
		for id, link := range res.NewPictures {
			pl = append(pl, file.VirtualPicture{Id: id, Link: link})
		}
		if err := file.Repo.SaveVirtualPic(pl); err != nil {
			log.Println("SaveVirtualPic Error: ", err)
		}
	}
	return res
}

// collectFilmMT 并发采集影片信息
func collectFilmMT(start, capacity int, s *collect.FilmSource, r util.RequestInfo, collectFunc func(s *collect.FilmSource, r util.RequestInfo)) {
	// 初始化 channel, 容量为 capacity
	ch := make(chan int, capacity)

	// 收集结束标识
	waitCh := make(chan int)
	// 循环将所有需采集的页码写入 ch
	for i := max(start, 1); i <= capacity; i++ {
		ch <- i
	}
	close(ch)
	// 开启 MAXGoroutine 数量的协程, 如果分页页数小于设定的最大线程数, 则将线程数设置为1
	var GoroutineNum = config.MAXGoroutine
	if capacity < GoroutineNum*5 {
		GoroutineNum = 1
	}
	// 如果满足开启并发的条件, 则开启GoroutineNum数量的协程进行并发采集
	for i := 0; i < GoroutineNum; i++ {
		go func() {
			defer func() { waitCh <- 0 }()
			for {
				// 从channel中获取 pageNumber
				pg, ok := <-ch
				if !ok {
					break
				}
				// 采集任务被中止时跳过剩余页码
				if stopRequested(s.Id) {
					continue
				}
				// 执行对应的采集方法, 并发时不同使用同一个requestInfo
				requestInfo := util.CopyRequestInfo(r)
				requestInfo.Params.Set("pg", fmt.Sprint(pg))
				collectFunc(s, requestInfo)
			}
		}()
	}
	// 等待所有协程执行完毕
	for i := 0; i < GoroutineNum; i++ {
		<-waitCh
	}
}

// BatchCollect 批量采集, 采集指定的所有站点最近x小时内更新的数据
func BatchCollect(h int, ids ...string) {
	for _, id := range ids {
		// 如果查询到对应Id的资源站信息, 且资源站处于启用状态
		if fs := collect.Repo.FindCollectSourceById(id); fs != nil && fs.State {
			// 采用协程并发执行, 每个站点单独开启一个协程执行
			go func() {
				err := HandleCollectRefine(fs.Id, h)
				if err != nil {
					log.Println(err)
				}
			}()
		}
	}
}

// AutoCollect 自动进行对所有已启用站点的采集任务
func AutoCollect(h int) {
	// 获取采集站中所有站点, 进行遍历
	for _, s := range collect.Repo.GetCollectSourceList() {
		// 如果当前站点为启用状态 则执行 HandleCollect 进行数据采集
		if s.State {
			if err := HandleCollectRefine(s.Id, h); err != nil {
				log.Println(err)
			}
		}
	}
}

// CollectSingleFilm 重新采集指定的本站影片 (vodIds 以逗号分隔): 按影片的来源, 向各采集接口重新请求
func CollectSingleFilm(vodIds string) {
	var ids []int64
	for _, v := range strings.Split(vodIds, ",") {
		if id, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil {
			ids = append(ids, id)
		}
	}
	for sourceId, origins := range film.OriginsOf(ids) {
		s := collect.Repo.FindCollectSourceById(sourceId)
		if s == nil || !s.State {
			continue
		}
		strIds := make([]string, len(origins))
		for i, id := range origins {
			strIds[i] = strconv.FormatInt(id, 10)
		}
		collectFilmById(strings.Join(strIds, ","), s)
	}
}

// collectFilmById 采集采集站中指定ID (采集站的影片ID, 逗号分隔) 的影片; 该接口没有正在进行的采集时单独记录进度
func collectFilmById(ids string, s *collect.FilmSource) {
	var runErr error
	if beginProgress(s, "按ID采集", TriggerManual, 0) == nil {
		trackProgress(s.Id, func(p *Progress) {
			p.TypeCount, p.TypeIndex, p.TypeName, p.PageCount, p.Total = 1, 1, "指定视频", 1, len(strings.Split(ids, ","))
		})
		defer func() { finishProgress(s.Id, runErr) }()
	}
	r := util.RequestInfo{Uri: s.Uri, Params: s.Query()}
	r.Params.Set("pg", "1")
	r.Params.Set("ids", ids)
	list, err := spiderCore.GetFilmDetail(r)
	if err != nil || len(list) <= 0 {
		log.Println("GetMovieDetail Error: ", err)
		runErr = fmt.Errorf("获取视频详情失败: %v", err)
		return
	}
	res := ingest(s, list)
	trackProgress(s.Id, func(p *Progress) {
		p.pageDone(1)
		p.Added, p.Updated, p.Skipped = p.Added+res.Added, p.Updated+res.Updated, p.Skipped+res.Skipped
		for _, it := range res.Items {
			if it.Reason != "" {
				p.logf("%s  %s (%s)", it.Name, it.Action, it.Reason)
			} else {
				p.logf("%s  %s", it.Name, it.Action)
			}
		}
	})
	if s.SyncPictures() {
		syncFilmPicture()
	}
	ClearCache()
}

// syncFilmPicture 同步新采集入栈还未同步的图片
//
// spider 是后台采集引擎(非 HTTP 调用路径), 与 collect.Repo / film.MovieRepo 一样
// 直接调用 file.Repo, 不经过 file.Service。
func syncFilmPicture() {
	pl := file.Repo.PopVirtualPics(config.MaxScanCount)
	if len(pl) <= 0 {
		return
	}
	for _, vp := range pl {
		if file.Repo.ExistFileInfoByRid(vp.Id) {
			continue
		}
		fileName, err := util.SaveOnlineFile(vp.Link, config.FilmPictureUploadDir)
		if err != nil {
			continue
		}
		file.Repo.SaveGallery(file.FileInfo{
			Link:        fmt.Sprint(config.FilmPictureUrlPath, fileName),
			Uid:         config.UserIdInitialVal,
			RelevanceId: vp.Id,
			Type:        0,
			Fid:         regexp.MustCompile(`\.[^.]+$`).ReplaceAllString(fileName, ""),
			FileType:    strings.TrimPrefix(filepath.Ext(fileName), "."),
		})
	}
	syncFilmPicture()
}

// ======================================================= 采集拓展内容  =======================================================

// ======================================================= 公共方法  =======================================================

// collectInterval 采集请求间隔 (ms): 采集接口自身的间隔优先, 未设置时使用后台「采集间隔」(秒)
func collectInterval(s *collect.FilmSource) int {
	if s.Interval > 0 {
		return s.Interval
	}
	return siteconfig.Svc.GetSiteBasicConfig().CollectInterval * 1000
}

// collectTypes 需要采集的采集站分类: 有 (含继承的) 绑定的采集站分类, 未绑定的分类不入库, 也不请求
func collectTypes(binds map[int64][]int64) []int64 {
	var types []int64
	for typeId, categoryIds := range binds {
		if len(categoryIds) > 0 {
			types = append(types, typeId)
		}
	}
	slices.Sort(types)
	return types
}

// 采集站的分类列表缓存 (分类很少变动, 缓存 10 分钟), 用于子分类继承一级分类的绑定与显示分类名称
var (
	classMu    sync.Mutex
	classCache = map[string]struct {
		classes []collect.FilmClass
		at      time.Time
	}{}
)

// sourceClasses 采集站的分类列表, 请求失败时返回 nil
func sourceClasses(s *collect.FilmSource) []collect.FilmClass {
	classMu.Lock()
	defer classMu.Unlock()
	if c, ok := classCache[s.Id]; ok && time.Since(c.at) < 10*time.Minute {
		return c.classes
	}
	classes, err := spiderCore.GetClassList(util.RequestInfo{Uri: s.Uri, Params: s.Query()})
	if err != nil {
		log.Printf("采集接口「%s」获取分类列表失败: %v", s.Name, err)
		return nil
	}
	classCache[s.Id] = struct {
		classes []collect.FilmClass
		at      time.Time
	}{classes, time.Now()}
	return classes
}

// classNames 采集站分类ID → 名称
func classNames(s *collect.FilmSource) map[int64]string {
	names := make(map[int64]string)
	for _, c := range sourceClasses(s) {
		names[int64(c.TypeID)] = c.TypeName
	}
	return names
}

// effectiveBinds 采集接口的有效分类绑定: 采集站分类自己的绑定, 没有单独绑定的子分类沿用其一级分类的绑定
func effectiveBinds(s *collect.FilmSource) map[int64][]int64 {
	binds := collect.Repo.GetBindMap(s.Id)
	eff := make(map[int64][]int64, len(binds))
	for id, l := range binds {
		eff[id] = l
	}
	for _, c := range sourceClasses(s) {
		id, pid := int64(c.TypeID), int64(c.TypePid)
		if len(eff[id]) == 0 && pid != 0 && len(binds[pid]) > 0 {
			eff[id] = binds[pid]
		}
	}
	return eff
}

// applyCategoryBind 按分类绑定设置影片所属的本站分类 (Types, 可跨分类方案); 主要分类 (Cid/Pid) 优先取默认方案中的绑定分类.
// 未绑定或绑定的分类都已不存在时 Types 为空, 入库时跳过
func applyCategoryBind(binds map[int64][]int64, list []film.MovieDetail) {
	if len(binds) == 0 {
		return
	}
	categories := make(map[int64]film.Category)
	for _, c := range film.CategoryRepo.List(film.CategoryVideo, 0) {
		categories[c.Id] = c
	}
	for i := range list {
		var types []int64
		var primary *film.Category
		for _, id := range binds[list[i].Cid] {
			c, ok := categories[id]
			if !ok {
				continue
			}
			types = append(types, c.Id)
			if primary == nil || (primary.SchemeId != film.DefaultSchemeId && c.SchemeId == film.DefaultSchemeId) {
				primary = &c
			}
		}
		if primary == nil {
			continue
		}
		list[i].Types = types
		list[i].Cid, list[i].CName = primary.Id, primary.Name
		if list[i].Pid = primary.Pid; primary.Pid == 0 {
			list[i].Pid = primary.Id
		}
	}
}

// BrowseSource 浏览采集站资源列表 (ac=list), 可按采集站分类 t / 关键字 wd 筛选, 不写入任何数据
func BrowseSource(s collect.FilmSource, t int64, wd string, pg int) (collect.FilmListPage, error) {
	var page collect.FilmListPage
	if s.ResultModel != collect.JsonResult {
		return page, errors.New("暂不支持浏览 XML 类型的采集接口")
	}
	r := util.RequestInfo{Uri: s.Uri, Params: s.Query()}
	r.Params.Set("ac", "list")
	r.Params.Set("pg", fmt.Sprint(max(pg, 1)))
	if t > 0 {
		r.Params.Set("t", fmt.Sprint(t))
	}
	if wd != "" {
		r.Params.Set("wd", wd)
	}
	util.ApiGet(&r)
	if len(r.Resp) == 0 {
		return page, errors.New("采集接口请求失败, 返回数据为空")
	}
	if err := json.Unmarshal(r.Resp, &page); err != nil {
		return page, errors.New(fmt.Sprint("采集接口返回数据异常: ", err))
	}
	return page, nil
}

// CollectApiTest 测试采集接口是否可用: 按表单中的接口地址、附加参数、接口类型与资源类型请求一页数据并解析, 不写入任何数据
func CollectApiTest(s collect.FilmSource) error {
	// 使用当前采集站接口采集一页数据
	r := util.RequestInfo{Uri: s.Uri, Params: s.Query()}
	r.Params.Set("ac", s.CollectType.GetActionType())
	r.Params.Set("pg", "3")
	if err := util.ApiTest(&r); err != nil {
		return errors.New(fmt.Sprint("测试失败, 请求响应异常 : ", err.Error()))
	}
	switch s.ResultModel {
	case collect.JsonResult:
		// 只校验通用分页结构, 各资源类型的 list 字段不同
		var cp = collect.CommonPage{}
		if err := json.Unmarshal(r.Resp, &cp); err != nil {
			return errors.New(fmt.Sprint("测试失败, 返回数据异常, JSON序列化失败: ", err))
		}
		return nil
	case collect.XmlResult:
		var rd = collect.RssD{}
		if err := xml.Unmarshal(r.Resp, &rd); err != nil {
			return errors.New(fmt.Sprint("测试失败, 返回数据异常, XML序列化失败", err))
		}
		return nil
	}
	return errors.New("测试失败, 接口返回值类型不符合规范")
}
