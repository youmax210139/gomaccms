package request

import (
	"gomaccms/internal/seo"
	"gomaccms/internal/siteconfig"
)

// SiteText 站点的文字 (在 SEO 弹窗中按语言编辑): 网站名称、SEO、法律信息及其各语言译文
type SiteText struct {
	SiteName  string `json:"siteName"`
	SeoTitle  string `json:"seoTitle"`
	Keyword   string `json:"keyword"`
	Describe  string `json:"describe"`
	LegalInfo string `json:"legalInfo"`
	// I18n 各语言的网站名称、SEO 与法律信息 (留空的字段前台使用原文)
	I18n map[string]siteconfig.SiteText `json:"i18n"`
}

// SiteInfo 站点展示信息 (全局基础参数与每个域名共用的字段) = 文字 + 不分语言的设定
type SiteInfo struct {
	SiteText
	Logo          string `json:"logo"`
	ServiceEmail  string `json:"serviceEmail"`
	AnalyticsCode string `json:"analyticsCode"`
}

// SiteSEORequest SEO 弹窗保存: ID 为 0 时为「默认」站点, 否则为该域名
type SiteSEORequest struct {
	SiteText
	ID uint `json:"id"`
}

// UpdateSiteBasicRequest 基本设置 (关闭提示 + 后台设置) 更新请求参数
type UpdateSiteBasicRequest struct {
	Hint string `json:"hint"`
	// HintI18n 各语言的关闭提示 (留空时前台使用原文)
	HintI18n        map[string]string `json:"hintI18n"`
	CollectInterval int               `json:"collectInterval"`
	PageSize        int               `json:"pageSize"`
	LoginCaptcha    bool              `json:"loginCaptcha"`
	DuplicateRule   []string          `json:"duplicateRule"`
}

// UpdateDefaultSiteRequest 「默认」站点 (未配置的域名) 更新请求参数
type UpdateDefaultSiteRequest struct {
	SiteInfo
	Remark string `json:"remark"`
	State  bool   `json:"state"`
}

// DomainStateRequest 网站状态切换请求参数, ID 为 0 表示「默认」站点
type DomainStateRequest struct {
	ID    uint `json:"id"`
	State bool `json:"state"`
}

// DomainRequest 域名新增/更新请求参数: 域名 → 主题映射及该域名独立的站点信息
type DomainRequest struct {
	SiteInfo
	ID       uint   `json:"id"`
	Domain   string `json:"domain"`
	Theme    string `json:"theme"`
	State    bool   `json:"state"`
	SchemeId int64  `json:"schemeId"`
}

// BannerRequest 海报新增 (Id 为 0) / 修改, 或切换启用状态
type BannerRequest struct {
	Id       int64                            `json:"id"`
	SchemeId int64                            `json:"schemeId"`
	Mid      int64                            `json:"mid"`
	Name     string                           `json:"name"`
	Poster   string                           `json:"poster"`
	Picture  string                           `json:"picture"`
	Sort     int64                            `json:"sort"`
	Status   bool                             `json:"status"`
	I18n     map[string]siteconfig.BannerText `json:"i18n"`
}

// SEORulesRequest 分类方案的 SEO 规则 (页面类型 → 语言 → title / keywords / description), 整份替换
type SEORulesRequest struct {
	SchemeId int64     `json:"schemeId"`
	Rules    seo.Rules `json:"rules"`
}

// AdRequest 广告新增 (Id 为 0) / 修改, 或切换启用状态
type AdRequest struct {
	Id       int64  `json:"id"`
	SchemeId int64  `json:"schemeId"`
	Slot     string `json:"slot"`
	Name     string `json:"name"`
	Image    string `json:"image"`
	Link     string `json:"link"`
	Sort     int64  `json:"sort"`
	Status   bool   `json:"status"`
}
