package index

import (
	"encoding/json"
	"fmt"
	"gomaccms/internal/config"
	"gomaccms/internal/film"
	"gomaccms/internal/paging"
	"gomaccms/internal/siteconfig"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

var Svc *Service

type Service struct{}

// indexContentItem 首页单个分类展示区块的结构, 与 IndexPage 缓存未命中时
// 拼出的 map[string]interface{}{"nav":..., "movies":...} 形状一致,
// 仅用于 rehydrateIndexPage 把缓存命中拿到的通用 map 重新解码回具体类型。
type indexContentItem struct {
	Nav    *film.CategoryTree    `json:"nav"`
	Movies []film.MovieBasicInfo `json:"movies"`
}

// rehydrateIndexPage 将 film.SearchRepo.GetCacheData 缓存命中时返回的
// map[string]interface{}(其内部值经过一次 json.Marshal/Unmarshal 后已经退化为
// map[string]interface{}/[]interface{} 等通用类型, 字段名也变成了 json tag 的
// 小写形式) 重新序列化再按 IndexPage 缓存未命中分支实际构造的具体类型解码回去,
// 使得无论缓存命中与否, IndexPage 返回的 "category"/"content"/"banners" 三个
// key 下面挂的值类型都一致 —— 这正是 html/template 通过反射按 PascalCase
// 字段名取值(如 SSR 首页模板里的 {{$b.Mid}}/{{.nav.Show}})所要求的。
// 解码失败时返回 nil, 调用方应当当作缓存未命中处理, 直接回源重新计算。
func rehydrateIndexPage(info map[string]interface{}) map[string]interface{} {
	raw, err := json.Marshal(info)
	if err != nil {
		return nil
	}
	var typed struct {
		Category film.CategoryTree  `json:"category"`
		Content  []indexContentItem `json:"content"`
		Banners  siteconfig.Banners `json:"banners"`
	}
	if err := json.Unmarshal(raw, &typed); err != nil {
		return nil
	}
	list := make([]map[string]interface{}, len(typed.Content))
	for i, c := range typed.Content {
		list[i] = map[string]interface{}{"nav": c.Nav, "movies": c.Movies}
	}
	return map[string]interface{}{"category": typed.Category, "content": list, "banners": typed.Banners}
}

// localizeIndex 首页数据按语言覆盖分类名与影片文字 (缓存保持原文)
func localizeIndex(info map[string]interface{}, lang string) map[string]interface{} {
	if tree, ok := info["category"].(film.CategoryTree); ok {
		tree.Localize(lang)
		info["category"] = tree
	}
	if list, ok := info["content"].([]map[string]interface{}); ok {
		for _, item := range list {
			if nav, ok := item["nav"].(*film.CategoryTree); ok {
				nav.Localize(lang)
			}
			if movies, ok := item["movies"].([]film.MovieBasicInfo); ok {
				film.LocalizeBasics(movies, lang)
			}
		}
	}
	return info
}

// IndexPage 首页数据处理; schemeId 为请求域名使用的分类方案, lang 为分类名使用的语言
func (s *Service) IndexPage(schemeId int64, lang string) map[string]interface{} {
	// 首页请求时长较高, 采用redis按分类方案分别缓存, 在定时任务更新影片时清除对应缓存
	cacheKey := config.HomeCacheKeyOf(schemeId)
	info := film.SearchRepo.GetCacheData(cacheKey)
	if info != nil {
		// GetCacheData 内部经过一次 json 反序列化到 map[string]interface{},
		// 丢失了原本的具体类型(film.CategoryTree/film.MovieBasicInfo/...),
		// 这里重新 rehydrate 回具体类型再返回, 否则 SSR 模板按结构体字段取值会
		// 全部落空、渲染出空白页面。rehydrate 失败时不返回半成品, 直接回源。
		if typed := rehydrateIndexPage(info); typed != nil {
			return localizeIndex(typed, lang)
		}
	}
	info = make(map[string]interface{})
	// 1. 首页分类数据处理 导航分类数据处理, 只提供 电影 电视剧 综艺 动漫 四大顶级分类和其子分类
	tree := film.CategoryTree{Category: &film.Category{Id: 0, Name: "分类信息"}}
	sysTree := film.CategoryRepo.GetSchemeCategoryTree(schemeId)
	for _, c := range sysTree.Children {
		if c.Show {
			tree.Children = append(tree.Children, c)
		}
	}
	info["category"] = tree
	// 2. 提供用于首页展示的顶级分类影片信息, 每分类 14条数据
	var list []map[string]interface{}
	for _, c := range tree.Children {
		page := paging.Page{PageSize: 14, Current: 1}
		var movies []film.MovieBasicInfo
		if c.Children != nil {
			movies = film.MovieRepo.GetMovieListByPid(c.Id, &page)
		} else {
			movies = film.MovieRepo.GetMovieListByCid(c.Id, &page)
		}
		item := map[string]interface{}{"nav": c, "movies": movies}
		list = append(list, item)
	}
	info["content"] = list
	// 3. 获取首页轮播数据
	info["banners"] = siteconfig.Svc.Banners(schemeId, true)
	film.SearchRepo.DataCache(cacheKey, info)
	return localizeIndex(info, lang)
}

// ClearIndexCache 删除首页数据缓存
func (s *Service) ClearIndexCache() {
	film.SearchRepo.RemoveCache(config.IndexCacheKey)
}

// GetFilmDetail 影片详情信息页面处理: 所属分类取影片在 schemeId 分类方案中的分类;
// 影片不在该方案中 (请求域名看不到) 时 ok 为 false
func (s *Service) GetFilmDetail(id int64, schemeId int64, lang string) (film.MovieDetailVo, bool) {
	detail := film.MovieRepo.GetDetailByMid(id)
	// 不存在或未审核的视频前台不可见
	if detail.Mid == 0 || !film.SearchRepo.IsPublished(id) {
		return film.MovieDetailVo{}, false
	}
	cid, pid, ok := film.SearchRepo.SchemeCategoryOf(id, schemeId)
	if !ok {
		return film.MovieDetailVo{}, false
	}
	detail.Cid, detail.Pid = cid, pid
	film.LocalizeDetail(&detail, lang)
	if c, err := film.CategoryRepo.Find(cid); err == nil {
		c.Localize(lang)
		detail.CName = c.Name
	}
	return film.ConvertMovieDetailVo(detail, s.playLines(&detail)), true
}

// GetNavCategory 获取导航分类信息 (请求域名使用的分类方案, 名称按 lang 覆盖)
func (s *Service) GetNavCategory(schemeId int64, lang string) []*film.Category {
	tree := film.CategoryRepo.GetSchemeCategoryTree(schemeId)
	var cl []*film.Category
	for _, c := range tree.Children {
		if c.Show {
			c.Category.Localize(lang)
			cl = append(cl, c.Category)
		}
	}
	return cl
}

// SearchFilmInfo 获取关键字匹配的影片信息 (只含请求域名的分类方案中的影片)
func (s *Service) SearchFilmInfo(key string, page *paging.Page, schemeId int64, lang string) []film.MovieBasicInfo {
	ids := film.SearchRepo.SearchFilmKeyword(key, page, schemeId, lang)
	list := film.MovieRepo.GetBasicInfoByIds(ids)
	film.LocalizeBasics(list, lang)
	return list
}

// GetPidCategory 获取pid对应的分类信息 (按 lang 本地化), 不属于请求域名的分类方案时返回 nil
func (s *Service) GetPidCategory(pid int64, schemeId int64, lang string) *film.CategoryTree {
	tree := film.CategoryRepo.GetSchemeCategoryTree(schemeId)
	for _, t := range tree.Children {
		if t.Id == pid {
			t.Localize(lang)
			return t
		}
	}
	return nil
}

// RelateMovie 根据当前影片信息匹配相关的影片 (按原文片名与标签匹配, 结果按 lang 显示)
func (s *Service) RelateMovie(detail film.MovieDetailVo, page *paging.Page, lang string) []film.MovieBasicInfo {
	search := film.SearchInfo{
		Cid: detail.Cid, Name: detail.Name, ClassTag: detail.ClassTag,
		Area: detail.Area, Language: detail.Language,
	}
	// detail 可能已经按语言覆盖, 匹配要用原文
	if src := film.SearchRepo.GetSearchInfoById(detail.Mid); src != nil {
		search.Name, search.ClassTag, search.Area, search.Language = src.Name, src.ClassTag, src.Area, src.Language
	}
	list := film.MovieRepo.GetRelateMovieBasicInfo(search, page)
	film.LocalizeBasics(list, lang)
	return list
}

// SearchTags 整合对应分类的搜索tag; 「类型」标签的名称按语言覆盖 (值仍为分类ID)
func (s *Service) SearchTags(pid int64, lang string) map[string]interface{} {
	res := film.SearchRepo.GetSearchTag(pid)
	tags, _ := res["tags"].(map[string]interface{})
	cats, _ := tags["Category"].([]map[string]string)
	if len(cats) == 0 {
		return res
	}
	names := map[string]string{}
	for _, c := range film.CategoryRepo.GetChildrenTree(pid) {
		c.Localize(lang)
		names[fmt.Sprint(c.Id)] = c.Name
	}
	for _, t := range cats {
		if n, ok := names[t["Value"]]; ok {
			t["Name"] = n
		}
	}
	return res
}

// playLines 影片的播放线路: 每个播放组一条, 名称与顺序取自播放器设置 (未设置的播放组显示代码、排在最后), 停用的播放器不显示
func (s *Service) playLines(detail *film.MovieDetail) []film.PlayLinkVo {
	players := film.PlayerMap()
	type line struct {
		vo   film.PlayLinkVo
		sort int64
	}
	var lines []line
	for i, g := range detail.PlayList {
		code := fmt.Sprintf("play%d", i+1)
		if i < len(detail.PlayFrom) && detail.PlayFrom[i] != "" {
			code = detail.PlayFrom[i]
		}
		p, ok := players[code]
		if ok && !p.Status {
			continue
		}
		l := line{vo: film.PlayLinkVo{Id: code, Name: code, LinkList: g}, sort: math.MaxInt64}
		if ok {
			l.vo.Name, l.sort = p.Name, p.Sort
		}
		lines = append(lines, l)
	}
	sort.SliceStable(lines, func(i, j int) bool { return lines[i].sort < lines[j].sort })
	l := make([]film.PlayLinkVo, len(lines))
	for i, ln := range lines {
		l[i] = ln.vo
	}
	return l
}

// GetFilmsByTags 通过searchTag 返回满足条件的分页影片信息
func (s *Service) GetFilmsByTags(st film.SearchTagsVO, page *paging.Page, lang string) []film.MovieBasicInfo {
	ids := film.SearchRepo.GetSearchInfosByTags(st, page)
	list := film.MovieRepo.GetBasicInfoByIds(ids)
	film.LocalizeBasics(list, lang)
	return list
}

// GetFilmClassify 通过Pid返回当前所属分类下的首页展示数据
func (s *Service) GetFilmClassify(pid int64, page *paging.Page, lang string) map[string]interface{} {
	res := make(map[string]interface{})
	for i, key := range []string{"news", "hits", "score"} {
		list := film.SearchRepo.GetMovieListBySort(i, pid, page)
		film.LocalizeBasics(list, lang)
		res[key] = list
	}
	return res
}

// TodayMovies 今天更新的影片 (pids 为前台可见的一级分类)
func (s *Service) TodayMovies(pids []int64, page *paging.Page, lang string) []film.MovieBasicInfo {
	list := film.SearchRepo.TodayMovies(pids, page)
	film.LocalizeBasics(list, lang)
	return list
}

// cacheEntry / cached 每个前台页面都要显示的少量数据 (侧栏今日更新数、热搜词) 在内存中缓存一会儿;
// 跨日后立即失效, 免得「今日」数据停留在前一天
type cacheEntry struct {
	v  any
	at time.Time
}

func cached[T any](m *sync.Map, key string, ttl time.Duration, load func() T) T {
	if v, ok := m.Load(key); ok {
		if e := v.(cacheEntry); time.Since(e.at) < ttl && e.at.YearDay() == time.Now().YearDay() {
			return e.v.(T)
		}
	}
	v := load()
	m.Store(key, cacheEntry{v: v, at: time.Now()})
	return v
}

// idsKey 一组分类ID的缓存键
func idsKey(ids []int64) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = fmt.Sprint(id)
	}
	return strings.Join(parts, ",")
}

var todayCounts, hotSearches sync.Map

// TodayCount 今天更新的影片数 (pids 为前台可见的一级分类), 缓存 1 分钟
func (s *Service) TodayCount(pids []int64) int64 {
	return cached(&todayCounts, idsKey(pids), time.Minute, func() int64 { return film.SearchRepo.TodayCount(pids) })
}

// HotSearch 搜索框下拉的「大家都在搜」(pids 为前台可见的一级分类), 缓存 5 分钟
func (s *Service) HotSearch(pids []int64, lang string) []string {
	return cached(&hotSearches, idsKey(pids)+"|"+lang, 5*time.Minute, func() []string { return film.SearchRepo.HotKeywords(pids, 10, lang) })
}

// RankBoard 排行榜的一个分类: 分类与其前 N 部影片
type RankBoard struct {
	Category *film.Category
	List     []film.SearchInfo
}

// RankBoards 各一级分类的排行榜 (by: hits / score / new), 每个分类前 limit 部
func (s *Service) RankBoards(nav []*film.Category, by string, limit int, lang string) []RankBoard {
	boards := make([]RankBoard, 0, len(nav))
	for _, c := range nav {
		list := film.SearchRepo.RankMovies(c.Id, by, limit)
		film.LocalizeSearchInfos(list, lang)
		boards = append(boards, RankBoard{Category: c, List: list})
	}
	return boards
}

// ResetCaches 清除内存中的今日更新数与热搜词缓存 (内容变动或后台刷新时)
func (s *Service) ResetCaches() {
	todayCounts.Clear()
	hotSearches.Clear()
}
