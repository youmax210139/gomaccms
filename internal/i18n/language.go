// Package i18n 前台多语系: 语言列表 (languages 表) 与请求语言的解析.
// 分类方案选择默认语言与启用的语言 (category_schemes.default_lang / langs), 访客可在方案启用的语言间切换 (cookie lang)
package i18n

import "time"

const (
	// SourceLang 原文语言: 采集内容的语言, 不能停用
	SourceLang = "zh-CN"
	// CookieName 访客选择的语言
	CookieName = "lang"
)

// Language 前台语言 (languages 表)
type Language struct {
	Code      string    `json:"code" gorm:"primaryKey"` // 如 zh-CN / vi, 即 <html lang>
	Name      string    `json:"name"`                   // 前台显示的名称, 用该语言本身书写 (如 Tiếng Việt)
	LibreCode string    `json:"libreCode"`              // 翻译服务 (LibreTranslate) 的语言代码, 供影片机器翻译使用
	Enabled   bool      `json:"enabled"`
	Sort      int64     `json:"sort"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

// TableName 语言表表名
func (Language) TableName() string {
	return "languages"
}
