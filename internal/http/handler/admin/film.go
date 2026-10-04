package admin

import (
	"fmt"
	"gomaccms/internal/ad"
	"gomaccms/internal/film"
	"gomaccms/internal/http/request"
	"gomaccms/internal/http/response"
	"gomaccms/internal/seo"
	"gomaccms/internal/siteconfig"
	"gomaccms/internal/spider"
	"gomaccms/internal/theme"
	"gomaccms/internal/view/inertia"
	"net/url"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	gonertia "github.com/romsar/gonertia/v3"
)

// FilmSearchPage 影片列表管理页(Inertia 渲染)
func FilmSearchPage(c *gin.Context) {
	renderFilmSearch(c, "")
}

// renderFilmSearch 按请求参数筛选并渲染视频数据页; errMsg 非空时作为错误提示
func renderFilmSearch(c *gin.Context, errMsg string) {
	s := film.SearchVo{Paging: pageFrom(c.Request.URL.Query())}
	s.Name = c.DefaultQuery("name", "")
	s.Pid, _ = strconv.ParseInt(c.DefaultQuery("pid", "0"), 10, 64)
	s.Cid, _ = strconv.ParseInt(c.DefaultQuery("cid", "0"), 10, 64)
	s.Plot = c.DefaultQuery("plot", "")
	s.Area = c.DefaultQuery("area", "")
	s.Language = c.DefaultQuery("language", "")
	if year := c.DefaultQuery("year", ""); year != "" {
		s.Year, _ = strconv.ParseInt(year, 10, 64)
	}
	s.Remarks = c.DefaultQuery("remarks", "")
	s.Player = c.DefaultQuery("player", "")
	s.Picture = c.DefaultQuery("picture", "")
	s.Sort = c.DefaultQuery("sort", "")
	s.Status = c.DefaultQuery("status", "")
	s.Level = c.DefaultQuery("level", "")
	s.Lock = c.DefaultQuery("lock", "")
	// 先按分类方案筛选, 分类选项也是该方案中的分类
	s.SchemeId, _ = strconv.ParseInt(c.DefaultQuery("schemeId", "0"), 10, 64)
	if !film.CategorySvc.SchemeExists(s.SchemeId) {
		s.SchemeId = film.DefaultSchemeId
	}
	if begin := c.DefaultQuery("beginTime", ""); begin != "" {
		if beginTime, e := time.ParseInLocation(time.DateTime, begin, time.Local); e == nil {
			s.BeginTime = beginTime.Unix()
		}
	}
	if end := c.DefaultQuery("endTime", ""); end != "" {
		if endTime, e := time.ParseInLocation(time.DateTime, end, time.Local); e == nil {
			s.EndTime = endTime.Unix()
		}
	}

	options := film.Svc.GetSearchOptions(s.SchemeId)
	sl := film.Svc.GetFilmPage(s)
	props := gonertia.Props{
		"params":  s,
		"list":    sl,
		"options": options,
	}
	withFormError(props, errMsg)
	_ = inertia.RenderManage(c, "Film/Film", props)
}

// FilmAddPage 手动添加影片页(Inertia 渲染)
func FilmAddPage(c *gin.Context) {
	tree := film.CategorySvc.GetFilmClassTree()
	var category []map[string]any
	for _, top := range tree.Children {
		for _, child := range top.Children {
			category = append(category, map[string]any{"id": child.Id, "pid": child.Pid, "name": child.Name})
		}
	}
	_ = inertia.RenderManage(c, "Film/FilmAdd", gonertia.Props{
		"category": category,
	})
}

// FilmAdd 手动添加影片(Inertia 渲染)
func FilmAdd(c *gin.Context) {
	var fr = request.FilmDetailRequest{}
	if err := c.ShouldBindJSON(&fr); err != nil {
		FilmAddPage(c)
		return
	}
	fd := film.FilmDetailVo{
		Id: fr.Id, Cid: fr.Cid, Pid: fr.Pid, Name: fr.Name, Picture: fr.Picture,
		PlayFrom: fr.PlayFrom, DownFrom: fr.DownFrom, PlayLink: fr.PlayLink, DownloadLink: fr.DownloadLink,
		SubTitle: fr.SubTitle, CName: fr.CName, EnName: fr.EnName, Initial: fr.Initial, ClassTag: fr.ClassTag,
		Actor: fr.Actor, Director: fr.Director, Writer: fr.Writer, Remarks: fr.Remarks, ReleaseDate: fr.ReleaseDate,
		Area: fr.Area, Language: fr.Language, Year: fr.Year, State: fr.State, UpdateTime: fr.UpdateTime,
		AddTime: fr.AddTime, DbId: fr.DbId, DbScore: fr.DbScore, Hits: fr.Hits, Content: fr.Content,
	}
	if err := film.Svc.SaveFilmDetail(fd); err != nil {
		tree := film.CategorySvc.GetFilmClassTree()
		var category []map[string]any
		for _, top := range tree.Children {
			for _, child := range top.Children {
				category = append(category, map[string]any{"id": child.Id, "pid": child.Pid, "name": child.Name})
			}
		}
		_ = inertia.RenderManage(c, "Film/FilmAdd", gonertia.Props{
			"category": category,
			"errors":   formError(fmt.Sprint("视频添加失败, 视频信息保存错误: ", err.Error())),
		})
		return
	}
	inertia.Redirect(c, "/manage/film/add")
}

// FilmDelete 删除影片检索信息(Inertia 渲染, 逻辑删除)
func FilmDelete(c *gin.Context) {
	idStr := c.DefaultQuery("id", "")
	if idStr != "" {
		if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
			_ = film.Svc.DelFilm(id)
		}
	}
	// 回到删除前的筛选条件与页码
	inertia.Back(c)
}

// FilmBatchUpdate 批量设置视频的推荐 / 审核 / 锁定 / 人气(Inertia 渲染)
func FilmBatchUpdate(c *gin.Context) {
	var req request.FilmBatchUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		inertia.Back(c)
		return
	}
	if err := film.SearchRepo.BatchUpdateVods(req.Ids, req.Field, req.Value); err != nil {
		// 带着原页面的筛选条件重新渲染, 显示错误
		if u, perr := url.Parse(c.Request.Referer()); perr == nil {
			c.Request.URL.RawQuery = u.RawQuery
		}
		renderFilmSearch(c, err.Error())
		return
	}
	spider.ClearCache()
	inertia.Back(c)
}

// FilmBatchDelete 批量删除影片(Inertia 渲染)
func FilmBatchDelete(c *gin.Context) {
	var req request.IdsRequest[int64]
	if err := c.ShouldBindJSON(&req); err == nil {
		for _, id := range req.Ids {
			_ = film.Svc.DelFilm(id)
		}
	}
	spider.ClearCache()
	inertia.Back(c)
}

//----------------------------------------------------影片分类处理----------------------------------------------------

// FilmClassTree 分类管理页, ?scheme= 选择分类方案 (默认为默认方案)
func FilmClassTree(c *gin.Context) {
	renderFilmClass(c, "")
}

// schemeTab 分类管理页的方案页签: 方案信息 + 正在使用它的域名
type schemeTab struct {
	film.CategoryScheme
	Domains []string `json:"domains"`
}

// schemeTabs 全部分类方案及其使用中的域名 (未配置的域名使用默认方案)
func schemeTabs() []schemeTab {
	used := map[int64][]string{film.DefaultSchemeId: {"未配置的域名"}}
	for _, d := range theme.Svc.AllDomains() {
		used[d.SchemeOrDefault()] = append(used[d.SchemeOrDefault()], d.Domain)
	}
	var tabs []schemeTab
	for _, sc := range film.CategorySvc.ListSchemes() {
		tabs = append(tabs, schemeTab{CategoryScheme: sc, Domains: used[sc.Id]})
	}
	return tabs
}

// currentScheme 当前操作的分类方案: 请求参数 scheme, 其次是来源页面 (POST 请求) 的 scheme, 默认为默认方案
func currentScheme(c *gin.Context) int64 {
	raw := c.Query("scheme")
	if raw == "" {
		if u, err := url.Parse(c.Request.Referer()); err == nil {
			raw = u.Query().Get("scheme")
		}
	}
	if id, err := strconv.ParseInt(raw, 10, 64); err == nil && film.CategorySvc.SchemeExists(id) {
		return id
	}
	return film.DefaultSchemeId
}

// renderFilmClass 渲染分类管理页 (当前方案中全部类型的分类); errMsg 非空时作为错误提示
func renderFilmClass(c *gin.Context, errMsg string) {
	scheme := currentScheme(c)
	props := gonertia.Props{
		"scheme":  scheme,
		"schemes": schemeTabs(),
		"tree":    film.CategorySvc.GetAllClassTree(scheme),
		"counts":  film.CategorySvc.FilmCounts(scheme),
		"types":   film.CategoryTypes,
		// 分类对话框的语言页签: 当前方案启用的非原文语言
		"classLanguages": schemeTranslations(scheme),
	}
	withFormError(props, errMsg)
	_ = inertia.RenderManage(c, "Film/FilmClass", props)
}

// filmClassDone 分类修改后的统一处理: 失败时带错误重新渲染, 成功时清除首页缓存并回到原页面 (保留方案页签)
func filmClassDone(c *gin.Context, err error) {
	if err != nil {
		renderFilmClass(c, err.Error())
		return
	}
	spider.ClearCache()
	inertia.Back(c)
}

// FindFilmClass 获取指定ID对应的影片分类信息
func FindFilmClass(c *gin.Context) {
	idStr := c.DefaultQuery("id", "")
	if idStr == "" {
		response.Failed("视频分类信息获取失败, 分类Id不能为空", c)
		return
	}
	// 转化id类型为int
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.Failed("视频分类信息获取失败, 参数分类Id格式异常", c)
		return
	}
	// 通过Id返回对应的分类信息
	class := film.CategorySvc.GetFilmClassById(id)
	if class == nil {
		response.Failed("视频分类信息获取失败, 分类信息不存在", c)
		return
	}
	response.Success(class, "分类信息查找成功", c)
}

// UpdateFilmClass 行内修改分类的状态/排序(Inertia 渲染)
func UpdateFilmClass(c *gin.Context) {
	var req request.FilmClassUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Id == 0 {
		renderFilmClass(c, "请求参数异常")
		return
	}
	var err error
	if req.Show != nil {
		err = film.CategorySvc.SetClassShow(req.Id, *req.Show)
	}
	if err == nil && req.Sort != nil {
		err = film.CategorySvc.SetClassSort(req.Id, *req.Sort)
	}
	filmClassDone(c, err)
}

// SaveFilmClass 新增/编辑分类(Inertia 渲染)
func SaveFilmClass(c *gin.Context) {
	var req request.FilmClassSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		renderFilmClass(c, "请求参数异常")
		return
	}
	filmClassDone(c, film.CategorySvc.SaveClass(&film.Category{
		Id: req.Id, Type: req.Type, Pid: req.Pid, Name: req.Name, Slug: req.Slug, Show: req.Show, Sort: req.Sort,
		SchemeId: req.SchemeId,
		I18n:     req.I18n,
	}))
}

// renderSchemes 方案操作失败: 重新渲染站群管理页 (方案在其「方案管理」页签中维护) 并显示错误
func renderSchemes(c *gin.Context, errMsg string) {
	renderSiteConfig(c, siteconfig.Svc.GetSiteBasicConfig(), errMsg)
}

// SaveCategoryScheme 新增 / 修改分类方案(Inertia 渲染), 成功后回到站群管理页
func SaveCategoryScheme(c *gin.Context) {
	var req request.CategorySchemeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		renderSchemes(c, "请求参数异常")
		return
	}
	sc := film.CategoryScheme{Id: req.Id, Name: req.Name, Sort: req.Sort, DefaultLang: req.DefaultLang, Langs: req.Langs}
	if err := film.CategorySvc.SaveScheme(&sc); err != nil {
		renderSchemes(c, err.Error())
		return
	}
	inertia.Redirect(c, "/manage/config/basic")
}

// DelCategoryScheme 删除分类方案及其全部分类(Inertia 渲染); 有域名在使用时不能删除
func DelCategoryScheme(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Query("id"), 10, 64)
	for _, d := range theme.Svc.AllDomains() {
		if d.SchemeOrDefault() == id {
			renderSchemes(c, fmt.Sprintf("域名 %s 正在使用该方案, 请先在「站群管理」中改用其他方案", d.Domain))
			return
		}
	}
	if err := film.CategorySvc.DeleteScheme(id); err != nil {
		renderSchemes(c, err.Error())
		return
	}
	_ = seo.Svc.Save(id, nil) // 方案的 SEO 规则与广告一并删除
	_ = ad.Svc.DeleteScheme(id)
	spider.ClearCache()
	inertia.Redirect(c, "/manage/config/basic")
}

// DelFilmClass 删除指定ID对应的影片分类(Inertia 渲染)
func DelFilmClass(c *gin.Context) {
	id, err := strconv.ParseInt(c.Query("id"), 10, 64)
	if err != nil {
		renderFilmClass(c, "分类Id格式异常")
		return
	}
	filmClassDone(c, film.CategorySvc.DelClass(id))
}

// BatchFilmClassDel 批量删除影片分类(Inertia 渲染)
func BatchFilmClassDel(c *gin.Context) {
	var req request.FilmClassBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Ids) == 0 {
		renderFilmClass(c, "请先选择分类")
		return
	}
	for _, id := range req.Ids {
		// 删除一级分类时其子分类会一并删除, 之后再删除这些子分类会找不到, 忽略即可
		_ = film.CategorySvc.DelClass(id)
	}
	filmClassDone(c, nil)
}

// BatchFilmClassState 批量修改影片分类的显示状态(Inertia 渲染)
func BatchFilmClassState(c *gin.Context) {
	var req request.FilmClassBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Ids) == 0 {
		renderFilmClass(c, "请先选择分类")
		return
	}
	for _, id := range req.Ids {
		if err := film.CategorySvc.SetClassShow(id, req.Show); err != nil {
			filmClassDone(c, err)
			return
		}
	}
	filmClassDone(c, nil)
}

// TransferFilmClass 将所选分类下的影片转移到目标二级分类(Inertia 渲染)
func TransferFilmClass(c *gin.Context) {
	var req request.FilmClassBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Ids) == 0 || req.Target == 0 {
		renderFilmClass(c, "请先选择分类与目标分类")
		return
	}
	filmClassDone(c, film.CategorySvc.TransferFilms(req.Ids, req.Target))
}

// vodCategoryOptions 视频编辑页的主分类选项: 默认方案的视频分类 (一级与二级)
func vodCategoryOptions() []map[string]any {
	var out []map[string]any
	for _, top := range film.CategorySvc.GetFilmClassTree().Children {
		out = append(out, map[string]any{"id": top.Id, "name": top.Name})
		for _, child := range top.Children {
			out = append(out, map[string]any{"id": child.Id, "name": top.Name + " / " + child.Name})
		}
	}
	return out
}

// renderFilmEdit 渲染视频编辑页; errMsg 非空时作为错误提示
func renderFilmEdit(c *gin.Context, e film.VodEdit, errMsg string) {
	schemes := film.VodSchemes(e.Id)
	if len(schemes) == 0 {
		schemes = []int64{film.DefaultSchemeId}
	}
	_ = inertia.RenderManage(c, "Film/FilmEdit", withFormError(gonertia.Props{
		"vod":        e,
		"categories": vodCategoryOptions(),
		"players":    film.ListPlayers(),
		"languages":  schemeTranslations(schemes...),
	}, errMsg))
}

// FilmEditPage 视频编辑页 (Inertia 渲染), ?id= 为视频ID
func FilmEditPage(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Query("id"), 10, 64)
	e, err := film.Svc.LoadVodEdit(id)
	if err != nil {
		inertia.Redirect(c, "/manage/film/search/list")
		return
	}
	renderFilmEdit(c, e, "")
}

// FilmEdit 保存视频编辑, 成功后回到编辑页
func FilmEdit(c *gin.Context) {
	var e film.VodEdit
	if err := c.ShouldBindJSON(&e); err != nil {
		inertia.Back(c)
		return
	}
	if err := film.Svc.SaveVodEdit(e); err != nil {
		renderFilmEdit(c, e, err.Error())
		return
	}
	inertia.Redirect(c, fmt.Sprintf("/manage/film/edit?id=%d", e.Id))
}
