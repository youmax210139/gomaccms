package siteconfig

import "strings"

// SiteInfo 前台展示用的站点信息; 全局配置 (BasicConfig) 与每个域名 (theme.Domain)
// 各有一份, 请求域名在 domains 表中配置时整体使用该域名的 SiteInfo.
// Keyword / Describe 即 SEO Keywords / SEO Description
type SiteInfo struct {
	SiteName      string `json:"siteName" gorm:"size:50"`
	Logo          string `json:"logo" gorm:"size:500"`
	SeoTitle      string `json:"seoTitle" gorm:"size:100"`
	Keyword       string `json:"keyword" gorm:"size:200"`
	Describe      string `json:"describe" gorm:"column:seo_description;size:500"`
	ServiceEmail  string `json:"serviceEmail" gorm:"size:100"`
	AnalyticsCode string `json:"analyticsCode" gorm:"type:text"`
	LegalInfo     string `json:"legalInfo" gorm:"type:text"`
	// I18n 各语言的网站名称、SEO 与法律信息, 留空的字段使用原文 (Logo、Email、统计代码不分语言)
	I18n map[string]SiteText `json:"i18n" gorm:"serializer:json"`
}

// SiteText 站点信息在某个语言的文字
type SiteText struct {
	SiteName  string `json:"siteName"`
	SeoTitle  string `json:"seoTitle"`
	Keyword   string `json:"keyword"`
	Describe  string `json:"describe"`
	LegalInfo string `json:"legalInfo"`
}

// Localized lang 语言的站点信息: 译文非空的字段覆盖原文 (返回副本, 不修改原值)
func (s SiteInfo) Localized(lang string) SiteInfo {
	t, ok := s.I18n[lang]
	if !ok {
		return s
	}
	for _, f := range []struct {
		dst *string
		v   string
	}{
		{&s.SiteName, t.SiteName}, {&s.SeoTitle, t.SeoTitle}, {&s.Keyword, t.Keyword},
		{&s.Describe, t.Describe}, {&s.LegalInfo, t.LegalInfo},
	} {
		if f.v != "" {
			*f.dst = f.v
		}
	}
	return s
}

// NormalizeSiteI18n 去除首尾空白, 删除全部字段为空的语言
func NormalizeSiteI18n(m map[string]SiteText) map[string]SiteText {
	out := map[string]SiteText{}
	for lang, t := range m {
		t = SiteText{SiteName: strings.TrimSpace(t.SiteName), SeoTitle: strings.TrimSpace(t.SeoTitle),
			Keyword: strings.TrimSpace(t.Keyword), Describe: strings.TrimSpace(t.Describe), LegalInfo: strings.TrimSpace(t.LegalInfo)}
		if t != (SiteText{}) {
			out[lang] = t
		}
	}
	return out
}

// NormalizeHintI18n 去除首尾空白, 删除空的语言
func NormalizeHintI18n(m map[string]string) map[string]string {
	out := map[string]string{}
	for lang, h := range m {
		if h = strings.TrimSpace(h); h != "" {
			out[lang] = h
		}
	}
	return out
}

// BasicConfig 全局配置 (存于 Redis):
//   - SiteInfo / Remark / State: 「默认」站点, 用于未在 domains 表中配置的域名 (后台「网域与 Theme」的默认行);
//     已配置的域名使用各自的 SiteInfo 与 State
//   - Hint: 网站关闭时的提示, 所有域名共用
//   - CollectInterval / PageSize / LoginCaptcha: 后台设置
//
// Domain 已由 domains 表 (theme 包) 取代, 后台不再编辑, 仅为兼容旧数据保留
type BasicConfig struct {
	SiteInfo
	Domain string `json:"domain"`
	Remark string `json:"remark"`
	State  bool   `json:"state"`
	Hint   string `json:"hint"`
	// HintI18n 各语言的关闭提示, 没有时使用 Hint
	HintI18n        map[string]string `json:"hintI18n"`
	CollectInterval int               `json:"collectInterval"` // 采集间隔 (秒), 采集接口未单独设置间隔时使用, 0 为不限制
	PageSize        int               `json:"pageSize"`        // 后台列表每页条数
	LoginCaptcha    bool              `json:"loginCaptcha"`    // 后台登录是否需要图形验证码
	// DuplicateRule 入库重复规则: 不同采集接口的影片按这些栏位判断是否为同一部 (name 必选), 为空时使用默认规则
	DuplicateRule []string `json:"duplicateRule"`
}

// HintOf lang 语言的关闭提示, 没有译文时为原文
func (c BasicConfig) HintOf(lang string) string {
	if h := c.HintI18n[lang]; h != "" {
		return h
	}
	return c.Hint
}

// 入库重复规则可选的栏位
const (
	RuleName   = "name"   // 片名
	RuleYear   = "year"   // 年份 (影片没有年份时不比对)
	RuleType   = "type"   // 主要分类
	RuleArea   = "area"   // 地区
	RuleDouban = "douban" // 豆瓣ID (影片没有豆瓣ID时不比对)
)

// DuplicateRuleFields 入库重复规则可选的栏位及名称
var DuplicateRuleFields = map[string]string{RuleName: "名称", RuleYear: "年份", RuleType: "分类", RuleArea: "地区", RuleDouban: "豆瓣ID"}

// DefaultDuplicateRule 默认的入库重复规则: 名称 + 年份
var DefaultDuplicateRule = []string{RuleName, RuleYear}

// Rule 入库重复规则, 未设置时为默认规则
func (c BasicConfig) Rule() []string {
	if len(c.DuplicateRule) == 0 {
		return DefaultDuplicateRule
	}
	return c.DuplicateRule
}

// DefaultAdminPageSize 未设置「后台每页数」时的默认值
const DefaultAdminPageSize = 20

// AdminPageSize 后台列表每页条数, 未设置时使用默认值
func (c BasicConfig) AdminPageSize() int {
	if c.PageSize <= 0 {
		return DefaultAdminPageSize
	}
	return c.PageSize
}
