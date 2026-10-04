package public

import (
	"net/http"
	"strconv"
	"strings"

	"gomaccms/internal/film"
	"gomaccms/internal/i18n"
	"gomaccms/internal/index"
	"gomaccms/internal/seo"
	"gomaccms/internal/siteconfig"
	"gomaccms/internal/theme"
	"gomaccms/internal/view/maccms"
	"gomaccms/internal/view/renderer"

	"github.com/gin-gonic/gin"
)

// RenderPage 通过已解析的主题渲染前台页面, 并自动注入站点基础配置与顶级导航分类
// (每个页面的头部/底部公共模块都需要这些数据, 调用方无需各自单独获取)
func RenderPage(c *gin.Context, page string, data map[string]any) {
	if data == nil {
		data = map[string]any{}
	}
	global := siteconfig.Svc.GetSiteBasicConfig()
	site := localizeSite(withDomainSiteInfo(c, global), langOf(c))
	data["site"] = site
	nav := guestNav(index.Svc.GetNavCategory(schemeId(c), langOf(c)))
	data["nav"] = nav
	ids := navIds(nav)
	data["todayCount"] = index.Svc.TodayCount(ids)
	data["hotSearch"] = index.Svc.HotSearch(ids, langOf(c))
	data["seo"] = resolveSEO(c, site, data)
	// 可收录页面的 canonical (与 sitemap 中的 URL 一致)
	if p, ok := data["canonicalPath"].(string); ok {
		data["canonical"] = baseURL(c) + p
	}
	lang, langs := requestLang(c)
	data["lang"], data["langs"], data["langName"] = lang, langs, langName(langs, lang)
	withMacCMS(c, page, site, data)
	// 网站关闭时 (已配置的域名看该域名的状态, 其余看全局默认状态) 所有前台页面统一渲染主题的 closed 页,
	// 展示全局的关闭提示; 全局网站名称为必填项, 为空说明全局配置未加载 (如 Redis 数据丢失), 此时不视为关闭
	if !site.State && global.SiteName != "" {
		c.Status(http.StatusServiceUnavailable)
		page = "closed"
	}
	renderer.Render(c, page, data)
}

// PageSEO 页面的 <title> 与 SEO meta; 页面可通过 data["seo"] 提供自己的值 (如分类页), 空字段回退到站点配置
type PageSEO struct {
	Title       string
	Keywords    string
	Description string
}

func pageSEO(site siteconfig.BasicConfig, override any) PageSEO {
	seo := PageSEO{Title: site.SeoTitle, Keywords: site.Keyword, Description: site.Describe}
	if seo.Title == "" {
		seo.Title = site.SiteName
	}
	if o, ok := override.(PageSEO); ok {
		if o.Title != "" {
			seo.Title = o.Title
		}
		if o.Keywords != "" {
			seo.Keywords = o.Keywords
		}
		if o.Description != "" {
			seo.Description = o.Description
		}
	}
	return seo
}

// seoPageKey 页面提供 SEO 资料 (seoPage) 的数据键; 没有时页面用 data["seo"] 或站点 SEO
const seoPageKey = "seoPage"

// seoPage 页面的 SEO 资料: 占位符的值
type seoPage struct {
	ctx seo.Context
}

// resolveSEO 页面的 SEO: 有 seoPage 时按 方案的 SEO 规则 (页面类型 + 请求语言) → 站点 SEO 组合;
// 否则为页面给的 data["seo"] 覆盖站点 SEO (今日更新、排行榜等)
func resolveSEO(c *gin.Context, site siteconfig.BasicConfig, data map[string]any) PageSEO {
	siteSEO := pageSEO(site, nil)
	sp, ok := data[seoPageKey].(seoPage)
	if !ok {
		return pageSEO(site, data["seo"])
	}
	sp.ctx.Site = seo.Site{Name: site.SiteName, URL: baseURL(c), Keywords: site.Keyword, Description: site.Describe}
	data[seoPageKey] = sp // $obj / $param 使用补上站点后的资料
	rule := seo.Svc.Rule(schemeId(c), sp.ctx.PageType, langOf(c))
	t := seo.Build(sp.ctx, rule, seo.Text{Title: siteSEO.Title, Keywords: siteSEO.Keywords, Description: siteSEO.Description})
	return PageSEO{Title: t.Title, Keywords: t.Keywords, Description: t.Description}
}

// typeSEO 分类页 / 筛选页的 SEO 资料 (分类已按请求语言覆盖)
func typeSEO(pageType string, c *film.Category, page int) seoPage {
	return seoPage{ctx: seo.Context{PageType: pageType, Page: page, Type: &seo.Type{Name: c.Name}}}
}

// vodSEO 详情 / 播放页的影片资料 (已按请求语言覆盖)
func vodSEO(d film.MovieDetailVo) *seo.Vod {
	return &seo.Vod{Name: d.Name, Sub: d.SubTitle, Year: d.Year, Area: d.Area, Lang: d.Language, Class: d.ClassTag,
		Actor: d.Actor, Director: d.Director, Remarks: d.Remarks, Score: d.DbScore, Content: d.Content}
}

// schemeId 当前请求域名使用的分类方案, 未配置的域名为默认方案;
// 需要 middleware.ResolveTheme 先把域名记录放进上下文
func schemeId(c *gin.Context) int64 {
	if v, ok := c.Get("domain"); ok {
		if d, ok := v.(*theme.Domain); ok {
			return d.SchemeOrDefault()
		}
	}
	return theme.DefaultSchemeId
}

// withDomainSiteInfo 返回当前请求域名的站点配置: 以全局配置 (关闭提示等) 为基础,
// 请求域名在 domains 表中配置时, 站点信息与网站状态整体替换为该域名自己的
// (需要 middleware.ResolveTheme 先把域名记录放进上下文)
func withDomainSiteInfo(c *gin.Context, site siteconfig.BasicConfig) siteconfig.BasicConfig {
	if v, ok := c.Get("domain"); ok {
		if d, ok := v.(*theme.Domain); ok && d != nil {
			site.SiteInfo = d.SiteInfo
			site.State = d.State
		}
	}
	return site
}

// localizeSite 站点信息与关闭提示按语言覆盖 (留空的字段使用原文)
func localizeSite(site siteconfig.BasicConfig, lang string) siteconfig.BasicConfig {
	site.SiteInfo = site.SiteInfo.Localized(lang)
	site.Hint = site.HintOf(lang)
	return site
}

// navIds 导航分类 (前台可见的一级分类) 的 ID
func navIds(nav []*film.Category) []int64 {
	ids := make([]int64, len(nav))
	for i, c := range nav {
		ids[i] = c.Id
	}
	return ids
}

// requestLang 请求使用的语言与前台可切换的语言 (由 middleware.ResolveTheme 放进上下文)
func requestLang(c *gin.Context) (string, []i18n.Language) {
	v, _ := c.Get("langs")
	langs, _ := v.([]i18n.Language)
	if langs == nil {
		langs = []i18n.Language{}
	}
	return c.GetString("lang"), langs
}

// langName 语言切换按钮上显示的当前语言名称
func langName(langs []i18n.Language, lang string) string {
	for _, l := range langs {
		if l.Code == lang {
			return l.Name
		}
	}
	return lang
}

// langOf 请求使用的语言 (middleware.ResolveTheme 放进上下文)
func langOf(c *gin.Context) string {
	return c.GetString("lang")
}

// macAid MacCMS 的页面编号 ({$maccms.aid}): 首页 1, 分类 / 影片库 11 / 12, 搜索 13, 详情 14, 播放 15, 其他 0
var macAid = map[string]int{"index": 1, "filmClassify": 11, "filmClassifySearch": 12, "search": 13, "filmDetail": 14, "play": 15}

// withMacCMS MacCMS 模板标签的全局变量 ($maccms) 与数据标签的上下文 (方案、语言、页码 ?page=、当前分类 ?Pid=)
func withMacCMS(c *gin.Context, page string, site siteconfig.BasicConfig, data map[string]any) {
	seo, _ := data["seo"].(PageSEO)
	data[maccms.GlobalKey] = map[string]any{
		"site_name": site.SiteName, "site_url": baseURL(c), "site_keywords": seo.Keywords,
		"site_description": seo.Description, "path": "/", "aid": macAid[page],
	}
	p, _ := strconv.Atoi(c.Query("page"))
	if p <= 0 {
		p, _ = strconv.Atoi(c.Query("current"))
	}
	typeId, _ := strconv.ParseInt(c.Query("Pid"), 10, 64)
	data[maccms.CtxKey] = maccms.Ctx{SchemeId: schemeId(c), Lang: langOf(c), Page: p, TypeId: typeId, BaseURL: baseURL(c)}
	// {$obj.type_name} / {$obj.vod_name} 当前分类或影片, {$param.page} 页码 (与 SEO 规则的占位符同一份资料)
	obj := map[string]any{}
	if sp, ok := data[seoPageKey].(seoPage); ok {
		for k, v := range sp.ctx.Values() {
			if strings.HasPrefix(k, "type_") || strings.HasPrefix(k, "vod_") || strings.HasPrefix(k, "episode_") {
				obj[k] = v
			}
		}
	}
	data["obj"], data["param"] = obj, map[string]any{"page": max(p, 1)}
}
