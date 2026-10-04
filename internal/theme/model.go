package theme

import (
	"gomaccms/internal/siteconfig"

	"gorm.io/gorm"
)

// Domain maps a request Host to the theme that should render it, together
// with that host's own site info (name, logo, SEO, ...), which replaces the
// global siteconfig.BasicConfig site info as a whole for that host.
type Domain struct {
	gorm.Model
	Domain string `json:"domain" gorm:"uniqueIndex"`
	Theme  string `json:"theme"`
	State  bool   `json:"state"` // 网站状态, 关闭时该域名的前台页面显示关闭提示
	// SchemeId 使用的分类方案 (category_schemes.id); 0 视为默认方案
	SchemeId int64 `json:"schemeId"`
	siteconfig.SiteInfo
}

// DefaultSchemeId 默认分类方案, 与 film.DefaultSchemeId 相同 (theme 不依赖 film 包)
const DefaultSchemeId int64 = 1

// SchemeOrDefault 域名使用的分类方案, 未设置时为默认方案
func (d *Domain) SchemeOrDefault() int64 {
	if d == nil || d.SchemeId == 0 {
		return DefaultSchemeId
	}
	return d.SchemeId
}

// TableName 设置域名映射表的表名
func (d *Domain) TableName() string {
	return "domains"
}
