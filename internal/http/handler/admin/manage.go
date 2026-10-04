package admin

import (
	"fmt"
	"gomaccms/internal/collect"
	"gomaccms/internal/config"
	"gomaccms/internal/film"
	"gomaccms/internal/http/request"
	"gomaccms/internal/i18n"
	"gomaccms/internal/seo"
	"gomaccms/internal/siteconfig"
	"gomaccms/internal/theme"
	"gomaccms/internal/view/inertia"
	"gomaccms/internal/view/renderer"
	"net/mail"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	gonertia "github.com/romsar/gonertia/v3"
)

func ManageIndex(c *gin.Context) {
	// 后台首页: 最近的采集记录 (含定时任务触发的采集)
	_ = inertia.RenderManage(c, "Index", gonertia.Props{
		"recentCollects": collect.Repo.CollectLogs("", 10),
	})
}

// ------------------------------------------------------ 站点基本配置 ------------------------------------------------------

// SiteBasicConfig 网站基本配置(渲染后台设置页: 基础参数 + 域名与主题)
func SiteBasicConfig(c *gin.Context) {
	renderSiteConfig(c, siteconfig.Svc.GetSiteBasicConfig(), "")
}

// renderSiteConfig 渲染站点设置页; basic 为表单回显数据, errMsg 非空时作为表单错误提示
func renderSiteConfig(c *gin.Context, basic any, errMsg string) {
	props := gonertia.Props{
		"basic":   basic,
		"domains": theme.Svc.AllDomains(),
		"themes":  renderer.ThemeNames(),
		// 方案管理页签: 分类方案 (含使用中的域名)、语言、各方案的分类数
		"schemes":      schemeTabs(),
		"languages":    i18n.Svc.List(),
		"schemeCounts": film.CategorySvc.SchemeCategoryCounts(),
		// 各方案的 SEO 规则与各页面类型的默认标题 (方案的「SEO」按钮)
		"seoRules":    schemeSEORules(),
		"seoDefaults": seoDefaults(),
		"rules":       siteconfig.DuplicateRuleFields,
	}
	withFormError(props, errMsg)
	_ = inertia.RenderManage(c, "System/SiteConfig", props)
}

// UpdateSiteBasic 更新基本设置: 关闭提示与后台设置 (Inertia 渲染)
func UpdateSiteBasic(c *gin.Context) {
	var req request.UpdateSiteBasicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		renderSiteConfig(c, siteconfig.Svc.GetSiteBasicConfig(), fmt.Sprint("请求参数异常:  ", err))
		return
	}
	req.Hint = strings.TrimSpace(req.Hint)
	req.HintI18n = siteconfig.NormalizeHintI18n(req.HintI18n)
	if msg := validateSiteBasic(req); msg != "" {
		renderSiteConfig(c, siteconfig.Svc.GetSiteBasicConfig(), msg)
		return
	}
	// 以已保存配置为基础, 只覆盖基本设置中的字段 (默认站点信息等保持不变)
	bc := siteconfig.Svc.GetSiteBasicConfig()
	bc.Hint, bc.HintI18n, bc.CollectInterval, bc.PageSize, bc.LoginCaptcha = req.Hint, req.HintI18n, req.CollectInterval, req.PageSize, req.LoginCaptcha
	bc.DuplicateRule = req.DuplicateRule
	if err := siteconfig.Svc.UpdateSiteBasic(bc); err != nil {
		renderSiteConfig(c, bc, fmt.Sprint("基本设置更新失败:  ", err))
		return
	}
	inertia.Redirect(c, "/manage/config/basic")
}

// validateSiteBasic 校验基本设置, 返回错误提示 (空串表示通过)
func validateSiteBasic(req request.UpdateSiteBasicRequest) string {
	switch {
	case req.Hint == "":
		return "关闭提示不能为空"
	case utf8.RuneCountInString(req.Hint) > 500:
		return "关闭提示 不能超过 500 个字符"
	case !knownLangs(req.HintI18n):
		return "关闭提示的语言无效"
	case req.CollectInterval < 0 || req.CollectInterval > 3600:
		return "采集间隔需在 0 - 3600 秒之间"
	case req.PageSize < 5 || req.PageSize > 200:
		return "后台每页数需在 5 - 200 之间"
	}
	for lang, h := range req.HintI18n {
		if utf8.RuneCountInString(h) > 500 {
			return fmt.Sprintf("关闭提示 (%s) 不能超过 500 个字符", lang)
		}
	}
	hasName := false
	for _, f := range req.DuplicateRule {
		if siteconfig.DuplicateRuleFields[f] == "" {
			return fmt.Sprintf("未知的入库重复规则栏位 %q", f)
		}
		hasName = hasName || f == siteconfig.RuleName
	}
	if !hasName {
		return "入库重复规则必须包含「名称」"
	}
	return ""
}

// UpdateDefaultSite 更新「默认」站点 (未配置的域名使用的站点信息与网站状态)
func UpdateDefaultSite(c *gin.Context) {
	var req request.UpdateDefaultSiteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		renderSiteConfig(c, siteconfig.Svc.GetSiteBasicConfig(), fmt.Sprint("请求参数异常:  ", err))
		return
	}
	bc := siteconfig.Svc.GetSiteBasicConfig()
	if msg := validateSiteSettings(&req.SiteInfo); msg != "" {
		renderSiteConfig(c, bc, msg)
		return
	}
	if utf8.RuneCountInString(req.Remark) > 500 {
		renderSiteConfig(c, bc, "网站备注 不能超过 500 个字符")
		return
	}
	if !req.State && bc.Hint == "" {
		renderSiteConfig(c, bc, "关闭网站前请先在「基本设置」填写关闭提示")
		return
	}
	// 文字栏位在 SEO 弹窗中编辑, 这里只更新设定
	bc.Logo, bc.ServiceEmail, bc.AnalyticsCode = req.Logo, req.ServiceEmail, req.AnalyticsCode
	bc.Remark, bc.State = req.Remark, req.State
	if err := siteconfig.Svc.UpdateSiteBasic(bc); err != nil {
		renderSiteConfig(c, bc, fmt.Sprint("默认站点更新失败:  ", err))
		return
	}
	inertia.Redirect(c, "/manage/config/basic")
}

// DomainState 开启 / 关闭网站: ID 为 0 时切换「默认」站点, 否则切换对应域名
func DomainState(c *gin.Context) {
	var req request.DomainStateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		renderSiteConfig(c, siteconfig.Svc.GetSiteBasicConfig(), fmt.Sprint("请求参数异常:  ", err))
		return
	}
	bc := siteconfig.Svc.GetSiteBasicConfig()
	if !req.State && bc.Hint == "" {
		renderSiteConfig(c, bc, "关闭网站前请先在「基本设置」填写关闭提示")
		return
	}
	var err error
	if req.ID == 0 {
		bc.State = req.State
		err = siteconfig.Svc.UpdateSiteBasic(bc)
	} else {
		err = theme.Svc.SetDomainState(req.ID, req.State)
	}
	if err != nil {
		renderSiteConfig(c, bc, fmt.Sprint("网站状态更新失败:  ", err))
		return
	}
	inertia.Redirect(c, "/manage/config/basic")
}

// validateSiteInfo 新增域名: 文字 (网站名称必填) 与设定都要校验
func validateSiteInfo(si *request.SiteInfo) string {
	if msg := validateSiteText(&si.SiteText); msg != "" {
		return msg
	}
	return validateSiteSettings(si)
}

// validateSiteSettings 清理并校验不分语言的站点设定 (Logo / 客服 Email), 返回错误提示 (空串表示通过);
// 统计代码原样保存, 不做 trim / 转义 / 截断
func validateSiteSettings(si *request.SiteInfo) string {
	si.Logo = strings.TrimSpace(si.Logo)
	si.ServiceEmail = strings.TrimSpace(si.ServiceEmail)
	switch {
	case utf8.RuneCountInString(si.Logo) > 500:
		return "网站 Logo 不能超过 500 个字符"
	case utf8.RuneCountInString(si.ServiceEmail) > 100:
		return "客服 Email 不能超过 100 个字符"
	}
	if si.ServiceEmail != "" {
		if addr, err := mail.ParseAddress(si.ServiceEmail); err != nil || addr.Address != si.ServiceEmail {
			return "客服 Email 格式错误"
		}
	}
	return ""
}

// siteTextLimits 站点文字各栏位的长度限制 (原文与各语言相同)
func siteTextLimits(siteName, seoTitle, keyword, describe, legalInfo string) []struct {
	name  string
	value string
	max   int
} {
	return []struct {
		name  string
		value string
		max   int
	}{{"网站名称", siteName, 50}, {"SEO Title", seoTitle, 100}, {"SEO Keywords", keyword, 200},
		{"SEO Description", describe, 500}, {"法律信息", legalInfo, 2000}}
}

// validateSiteText 清理并校验站点文字 (网站名称必填) 及其各语言译文
func validateSiteText(t *request.SiteText) string {
	t.SiteName = strings.TrimSpace(t.SiteName)
	t.SeoTitle = strings.TrimSpace(t.SeoTitle)
	t.Keyword = strings.TrimSpace(t.Keyword)
	t.Describe = strings.TrimSpace(t.Describe)
	for _, l := range siteTextLimits(t.SiteName, t.SeoTitle, t.Keyword, t.Describe, t.LegalInfo) {
		if utf8.RuneCountInString(l.value) > l.max {
			return fmt.Sprintf("%s 不能超过 %d 个字符", l.name, l.max)
		}
	}
	if t.SiteName == "" {
		return "网站名称不能为空"
	}
	t.I18n = siteconfig.NormalizeSiteI18n(t.I18n)
	if !knownLangs(t.I18n) {
		return "网站信息的语言无效"
	}
	for lang, tr := range t.I18n {
		for _, l := range siteTextLimits(tr.SiteName, tr.SeoTitle, tr.Keyword, tr.Describe, tr.LegalInfo) {
			if utf8.RuneCountInString(l.value) > l.max {
				return fmt.Sprintf("%s (%s) 不能超过 %d 个字符", l.name, lang, l.max)
			}
		}
	}
	return ""
}

// withSiteText 以 t 替换站点信息的文字栏位 (设定栏位不变)
func withSiteText(si siteconfig.SiteInfo, t request.SiteText) siteconfig.SiteInfo {
	si.SiteName, si.SeoTitle, si.Keyword, si.Describe, si.LegalInfo, si.I18n = t.SiteName, t.SeoTitle, t.Keyword, t.Describe, t.LegalInfo, t.I18n
	return si
}

// SiteSEO 保存 SEO 弹窗: 「默认」站点 (ID 0) 或域名的网站名称、SEO、法律信息及各语言译文
func SiteSEO(c *gin.Context) {
	var req request.SiteSEORequest
	if err := c.ShouldBindJSON(&req); err != nil {
		renderSiteConfig(c, siteconfig.Svc.GetSiteBasicConfig(), fmt.Sprint("请求参数异常:  ", err))
		return
	}
	bc := siteconfig.Svc.GetSiteBasicConfig()
	if msg := validateSiteText(&req.SiteText); msg != "" {
		renderSiteConfig(c, bc, msg)
		return
	}
	var err error
	if req.ID == 0 {
		bc.SiteInfo = withSiteText(bc.SiteInfo, req.SiteText)
		err = siteconfig.Svc.UpdateSiteBasic(bc)
	} else {
		err = theme.Svc.UpdateDomainText(req.ID, withSiteText(siteconfig.SiteInfo{}, req.SiteText))
	}
	if err != nil {
		renderSiteConfig(c, siteconfig.Svc.GetSiteBasicConfig(), fmt.Sprint("SEO 保存失败:  ", err))
		return
	}
	inertia.Redirect(c, "/manage/config/basic")
}

func toSiteInfo(si request.SiteInfo) siteconfig.SiteInfo {
	return withSiteText(siteconfig.SiteInfo{Logo: si.Logo, ServiceEmail: si.ServiceEmail, AnalyticsCode: si.AnalyticsCode}, si.SiteText)
}

// knownLangs 译文的语言都是「系统语言」中原文以外的语言
func knownLangs[T any](m map[string]T) bool {
	known := map[string]bool{}
	for _, l := range i18n.Svc.List() {
		known[l.Code] = l.Code != i18n.SourceLang
	}
	for lang := range m {
		if !known[lang] {
			return false
		}
	}
	return true
}

// ------------------------------------------------------ 域名与主题 ------------------------------------------------------

// DomainAdd 新增域名 (主题 + 独立站点信息)
func DomainAdd(c *gin.Context) {
	saveDomain(c, false)
}

// DomainUpdate 修改域名、对应主题或其站点信息
func DomainUpdate(c *gin.Context) {
	saveDomain(c, true)
}

func saveDomain(c *gin.Context, update bool) {
	var req request.DomainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		renderSiteConfig(c, siteconfig.Svc.GetSiteBasicConfig(), fmt.Sprint("请求参数异常:  ", err))
		return
	}
	if !renderer.HasTheme(req.Theme) {
		renderSiteConfig(c, siteconfig.Svc.GetSiteBasicConfig(), fmt.Sprintf("主题 %q 不存在或不可用", req.Theme))
		return
	}
	// 新增时文字与设定一起保存; 修改只更新设定 (文字在 SEO 弹窗中编辑)
	validate := validateSiteSettings
	if !update {
		validate = validateSiteInfo
	}
	if msg := validate(&req.SiteInfo); msg != "" {
		renderSiteConfig(c, siteconfig.Svc.GetSiteBasicConfig(), msg)
		return
	}
	if !req.State && siteconfig.Svc.GetSiteBasicConfig().Hint == "" {
		renderSiteConfig(c, siteconfig.Svc.GetSiteBasicConfig(), "关闭网站前请先在「基本设置」填写关闭提示")
		return
	}
	if req.SchemeId == 0 {
		req.SchemeId = film.DefaultSchemeId
	}
	if !film.CategorySvc.SchemeExists(req.SchemeId) {
		renderSiteConfig(c, siteconfig.Svc.GetSiteBasicConfig(), "分类方案不存在")
		return
	}
	d := &theme.Domain{Domain: theme.NormalizeDomain(req.Domain), Theme: req.Theme, State: req.State, SchemeId: req.SchemeId,
		SiteInfo: toSiteInfo(req.SiteInfo)}
	var err error
	if update {
		if req.ID == 0 {
			err = fmt.Errorf("域名记录ID异常")
		} else {
			d.ID = req.ID
			err = theme.Svc.UpdateDomain(d)
		}
	} else {
		err = theme.Svc.CreateDomain(d)
	}
	if err != nil {
		renderSiteConfig(c, siteconfig.Svc.GetSiteBasicConfig(), err.Error())
		return
	}
	// 域名改用其他分类方案后, 首页缓存 (按方案) 需要重建
	film.SearchRepo.RemoveCache(config.IndexCacheKey)
	inertia.Redirect(c, "/manage/config/basic")
}

// DomainDel 删除域名-主题映射
func DomainDel(c *gin.Context) {
	if id, err := strconv.ParseUint(c.Query("id"), 10, 64); err == nil && id > 0 {
		_ = theme.Svc.DeleteDomain(uint(id))
	}
	inertia.Redirect(c, "/manage/config/basic")
}

// ------------------------------------------------------ 轮播数据配置 ------------------------------------------------------

// schemeSEORules 全部分类方案的 SEO 规则: 方案ID → 页面类型 → 语言 → 文字
func schemeSEORules() map[int64]seo.Rules {
	out := map[int64]seo.Rules{}
	for _, sc := range film.CategorySvc.ListSchemes() {
		out[sc.Id] = seo.Svc.Rules(sc.Id)
	}
	return out
}

// seoDefaults 各页面类型没有设置标题规则时使用的标题
func seoDefaults() map[string]string {
	out := map[string]string{}
	for _, t := range seo.PageTypes {
		out[t] = seo.DefaultTitle(t)
	}
	return out
}

// SaveSEORules 保存分类方案的 SEO 规则 (整份替换), 成功后回到站群管理页
func SaveSEORules(c *gin.Context) {
	var req request.SEORulesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		renderSiteConfig(c, siteconfig.Svc.GetSiteBasicConfig(), "请求参数异常")
		return
	}
	if !film.CategorySvc.SchemeExists(req.SchemeId) {
		renderSiteConfig(c, siteconfig.Svc.GetSiteBasicConfig(), "分类方案不存在")
		return
	}
	known := map[string]bool{}
	for _, l := range i18n.Svc.List() {
		known[l.Code] = true
	}
	for _, langs := range req.Rules {
		for lang := range langs {
			if !known[lang] {
				renderSiteConfig(c, siteconfig.Svc.GetSiteBasicConfig(), fmt.Sprintf("语言 %q 不存在", lang))
				return
			}
		}
	}
	if err := seo.Svc.Save(req.SchemeId, req.Rules); err != nil {
		renderSiteConfig(c, siteconfig.Svc.GetSiteBasicConfig(), err.Error())
		return
	}
	inertia.Redirect(c, "/manage/config/basic")
}
