package public

import (
	"gomaccms/internal/film"
	"gomaccms/internal/http/response"
	"gomaccms/internal/index"
	"gomaccms/internal/member"
	"gomaccms/internal/paging"
	"gomaccms/internal/publish"
	"gomaccms/internal/seo"
	"gomaccms/internal/siteconfig"
	"gomaccms/internal/view/renderer"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// Index 首页(SSR 渲染)
func Index(c *gin.Context) {
	data := index.Svc.IndexPage(schemeId(c), langOf(c))
	data["canonicalPath"] = publish.HomePath
	data[seoPageKey] = seoPage{ctx: seo.Context{PageType: seo.PageHome}}
	RenderPage(c, "index", data)
}

// SiteBasicConfigJSON 网站基本配置(供公开首页 /config/basic 使用, 回 JSON)
func SiteBasicConfigJSON(c *gin.Context) {
	bc := localizeSite(withDomainSiteInfo(c, siteconfig.Svc.GetSiteBasicConfig()), langOf(c))
	// 各语言的译文不对外输出 (已按请求语言覆盖)
	bc.I18n, bc.HintI18n = nil, nil
	// 网站备注与后台设置仅供后台使用, 不对外公开
	bc.Remark = ""
	bc.CollectInterval, bc.PageSize, bc.LoginCaptcha = 0, 0, false
	response.Success(bc, "网站基本信息获取成功", c)
}

// CategoriesInfo 分类信息获取
func CategoriesInfo(c *gin.Context) {
	data := index.Svc.GetNavCategory(schemeId(c), langOf(c))
	if len(data) == 0 {
		response.Failed("暂无分类信息", c)
		return
	}
	response.Success(data, "分类信息获取成功", c)
}

// FilmDetail 影片详情信息查询(SSR 渲染)
func FilmDetail(c *gin.Context) {
	id, err := strconv.Atoi(c.Query("link"))
	if err != nil {
		c.String(http.StatusBadRequest, "请求异常,视频请求参数异常!!!")
		return
	}
	detail, ok := index.Svc.GetFilmDetail(int64(id), schemeId(c), langOf(c))
	if !ok {
		c.String(http.StatusNotFound, "视频不存在")
		return
	}
	if guestDenied(c, member.PermDetail, detail.Pid, detail.Cid) {
		return
	}
	page := paging.Page{Current: 0, PageSize: 14}
	relateMovie := index.Svc.RelateMovie(detail, &page, langOf(c))
	RenderPage(c, "filmDetail", map[string]any{
		seoPageKey:      seoPage{ctx: seo.Context{PageType: seo.PageDetail, Vod: vodSEO(detail), Type: &seo.Type{Name: detail.CName}}},
		"canonicalPath": publish.VodPath(detail.Mid),
		"detail":        detail,
		"relate":        relateMovie,
	})
}

// History 观看历史页面(SSR 渲染, 数据完全来自客户端 cookie, 后端不持有观看记录)
func History(c *gin.Context) {
	RenderPage(c, "history", map[string]any{"noindex": true})
}

// FilmPlayInfo 影视播放页(SSR 渲染)
func FilmPlayInfo(c *gin.Context) {
	// 影片ID 无效时按 0 查询, 由下方的「视频不存在」处理
	id, _ := strconv.Atoi(c.DefaultQuery("id", "0"))
	playFrom := c.DefaultQuery("source", "")
	episode, err := strconv.Atoi(c.DefaultQuery("episode", "0"))
	if err != nil {
		c.String(http.StatusBadRequest, "请求异常,暂无视频信息!!!")
		return
	}
	detail, ok := index.Svc.GetFilmDetail(int64(id), schemeId(c), langOf(c))
	if !ok {
		c.String(http.StatusNotFound, "视频不存在")
		return
	}
	if guestDenied(c, member.PermPlay, detail.Pid, detail.Cid) {
		return
	}
	if len(playFrom) <= 1 && len(detail.List) > 0 {
		playFrom = detail.List[0].Id
	}
	var currentPlay film.PlayItem
	for _, v := range detail.List {
		if v.Id == playFrom {
			currentPlay = v.LinkList[episode]
		}
	}
	page := paging.Page{Current: 0, PageSize: 14}
	relateMovie := index.Svc.RelateMovie(detail, &page, langOf(c))
	RenderPage(c, "play", map[string]any{
		seoPageKey: seoPage{ctx: seo.Context{PageType: seo.PagePlay, Vod: vodSEO(detail), Type: &seo.Type{Name: detail.CName},
			Episode: &seo.Episode{Name: currentPlay.Episode, Index: episode + 1}}},
		"detail":          detail,
		"current":         currentPlay,
		"currentPlayFrom": playFrom,
		"currentEpisode":  episode,
		"relate":          relateMovie,
	})
}

// SearchFilm 通过片名模糊匹配库存中的信息
func SearchFilm(c *gin.Context) {
	keyword := c.DefaultQuery("keyword", "")
	currStr := c.DefaultQuery("current", "1")
	current, _ := strconv.Atoi(currStr)
	page := paging.Page{PageSize: 10, Current: current}
	bl := index.Svc.SearchFilmInfo(strings.TrimSpace(keyword), &page, schemeId(c), langOf(c))
	if page.Total <= 0 {
		response.Failed("暂无相关视频信息", c)
		return
	}
	response.Success(gin.H{"list": bl, "page": page}, "视频搜索成功", c)
}

// Search 影片搜索页面(SSR 渲染) -- 独立于 /searchFilm 的 JSON 接口, 该接口保持不变
func Search(c *gin.Context) {
	keyword := strings.TrimSpace(c.Query("search"))
	currStr := c.DefaultQuery("current", "1")
	current, _ := strconv.Atoi(currStr)
	page := paging.Page{PageSize: 10, Current: current}
	var list []film.MovieBasicInfo
	if keyword != "" {
		list = index.Svc.SearchFilmInfo(keyword, &page, schemeId(c), langOf(c))
	}
	RenderPage(c, "search", map[string]any{
		seoPageKey: seoPage{ctx: seo.Context{PageType: seo.PageSearch, Page: page.Current}},
		"noindex":  true,
		"list":     list,
		"page":     page,
		"search":   keyword,
	})
}

// FilmTagSearch 通过tag获取满足条件的对应影片
func FilmTagSearch(c *gin.Context) {
	params := film.SearchTagsVO{}
	pidStr := c.DefaultQuery("Pid", "")
	cidStr := c.DefaultQuery("Category", "")
	yStr := c.DefaultQuery("Year", "")
	if pidStr == "" {
		c.String(http.StatusBadRequest, "缺少分类信息")
		return
	}
	params.Pid, _ = strconv.ParseInt(pidStr, 10, 64)
	params.Cid, _ = strconv.ParseInt(cidStr, 10, 64)
	params.Plot = c.DefaultQuery("Plot", "")
	params.Area = c.DefaultQuery("Area", "")
	params.Language = c.DefaultQuery("Language", "")
	params.Year, _ = strconv.ParseInt(yStr, 10, 64)
	params.Sort = c.DefaultQuery("Sort", "release_stamp")

	currentStr := c.DefaultQuery("current", "1")
	current, _ := strconv.Atoi(currentStr)
	page := paging.Page{PageSize: 49, Current: current}
	// GetPidCategory returns nil for a Pid absent from the category tree (e.g.
	// no category has been scraped/seeded yet, or a bad query param) -- guard
	// before dereferencing the embedded *film.Category, mirroring FilmClassify's
	// nil-safe handling of the same lookup. Without this, a nonexistent Pid
	// panics on the ".Category" field access before the template's own
	// "{{if .title}}" guard ever gets a chance to degrade gracefully.
	// 分类不存在或已停用时返回 404; 选择了二级分类时以二级分类的 SEO 信息为准
	category := index.Svc.GetPidCategory(params.Pid, schemeId(c), langOf(c))
	if category == nil || !category.Show {
		c.String(http.StatusNotFound, "分类不存在")
		return
	}
	title := category.Category
	// 选择了二级分类时以二级分类的 SEO 为准
	sp := typeSEO(seo.PageShow, category.Category, current)
	if params.Cid != 0 {
		sub := findChild(category, params.Cid)
		if sub == nil || !sub.Show {
			c.String(http.StatusNotFound, "分类不存在")
			return
		}
		sp = typeSEO(seo.PageShow, sub.Category, current)
	}
	if guestDenied(c, member.PermList, params.Pid, params.Cid) {
		return
	}
	// GetFilmsByTags mutates page.Total/page.PageCount via the pointer as a
	// side effect; hoisted into its own statement (rather than inlined in
	// the map literal below alongside "page": page) so that mutation is
	// guaranteed to happen-before the map is built, instead of relying on
	// Go's unspecified evaluation order within a single composite literal.
	list := index.Svc.GetFilmsByTags(params, &page, langOf(c))
	RenderPage(c, "filmClassifySearch", map[string]any{
		"title":  title,
		"list":   list,
		"search": index.Svc.SearchTags(params.Pid, langOf(c)),
		"params": map[string]string{
			"Pid":      pidStr,
			"Category": cidStr,
			"Plot":     params.Plot,
			"Area":     params.Area,
			"Language": params.Language,
			"Year":     yStr,
			"Sort":     params.Sort,
		},
		"page":     page,
		seoPageKey: sp,
	})
}

// findChild 查找一级分类下指定ID的二级分类
func findChild(parent *film.CategoryTree, id int64) *film.CategoryTree {
	for _, c := range parent.Children {
		if c.Id == id {
			return c
		}
	}
	return nil
}

// FilmClassify  影片分类首页数据展示
func FilmClassify(c *gin.Context) {
	pidStr := c.DefaultQuery("Pid", "")
	if pidStr == "" {
		c.String(http.StatusBadRequest, "主分类信息获取异常")
		return
	}
	pid, _ := strconv.ParseInt(pidStr, 10, 64)
	title := index.Svc.GetPidCategory(pid, schemeId(c), langOf(c))
	// 分类不存在或已停用时返回 404
	if title == nil || !title.Show {
		c.String(http.StatusNotFound, "分类不存在")
		return
	}
	if guestDenied(c, member.PermList, pid) {
		return
	}
	page := paging.Page{PageSize: 21, Current: 1}
	RenderPage(c, "filmClassify", map[string]any{
		"canonicalPath": publish.ClassifyPath(pid),
		"title":         title,
		"content":       index.Svc.GetFilmClassify(pid, &page, langOf(c)),
		seoPageKey:      typeSEO(seo.PageType, title.Category, 1),
	})
}

// Today 今日更新页 (SSR 渲染): 今天更新的影片, 可按一级分类筛选 (Pid)
func Today(c *gin.Context) {
	nav := guestNav(index.Svc.GetNavCategory(schemeId(c), langOf(c)))
	pids := navIds(nav)
	pid, _ := strconv.ParseInt(c.Query("Pid"), 10, 64)
	if pid != 0 {
		if !slices.Contains(pids, pid) {
			c.String(http.StatusNotFound, "分类不存在")
			return
		}
		pids = []int64{pid}
	}
	current, _ := strconv.Atoi(c.DefaultQuery("current", "1"))
	page := paging.Page{PageSize: 42, Current: max(current, 1)}
	list := index.Svc.TodayMovies(pids, &page, langOf(c))
	RenderPage(c, "today", map[string]any{
		"canonicalPath": publish.TodayPath,
		"navKey":        "today",
		"pid":           pid,
		"list":          list,
		"page":          page,
		"seo":           PageSEO{Title: renderer.T(c, "nav.today")},
	})
}

// rankTabs 排行榜页签 (名称见语言包 rank.tab.*)
var rankTabs = []string{"hits", "score", "new"}

// Rank 排行榜页 (SSR 渲染): 每个一级分类一个榜单, by 为排序方式
func Rank(c *gin.Context) {
	by := c.DefaultQuery("by", "hits")
	if !slices.Contains(rankTabs, by) {
		by = "hits"
	}
	nav := guestNav(index.Svc.GetNavCategory(schemeId(c), langOf(c)))
	RenderPage(c, "rank", map[string]any{
		"canonicalPath": publish.RankPath,
		"navKey":        "rank",
		"by":            by,
		"tabs":          rankTabs,
		"boards":        index.Svc.RankBoards(nav, by, 10, langOf(c)),
		"seo":           PageSEO{Title: renderer.T(c, "nav.rank")},
	})
}

// IndexCacheDel 删除首页缓存数据
func IndexCacheDel(c *gin.Context) {
	index.Svc.ClearIndexCache()
	response.SuccessOnlyMsg("首页缓存数据已清除!!!", c)
}
