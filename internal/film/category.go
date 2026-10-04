package film

import (
	"errors"
	"strings"
	"time"
)

// 分类类型 (categories.type)
const (
	CategoryVideo   = 1
	CategoryArticle = 2
	CategoryActor   = 3
	CategoryWebsite = 4
)

// CategoryTypes 后台可选的分类类型
var CategoryTypes = map[int]string{CategoryVideo: "视频", CategoryArticle: "文章", CategoryActor: "演员", CategoryWebsite: "网站"}

// DefaultSchemeId 默认分类方案 (未配置的域名、未选方案的域名使用; 采集站分类导入到这里), 不能删除
const DefaultSchemeId int64 = 1

// ErrNegativeSort 排序值 (分类、分类方案、播放器) 不能为负
var ErrNegativeSort = errors.New("排序值不能为负数")

// CategoryScheme 分类方案 (category_schemes 表): 一套分类树, 每个域名选用一个方案
type CategoryScheme struct {
	Id          int64     `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name"`
	Sort        int64     `json:"sort"`
	DefaultLang string    `json:"defaultLang"`                  // 前台默认语言 (languages.code)
	Langs       []string  `json:"langs" gorm:"serializer:json"` // 前台可切换的语言
	CreatedAt   time.Time `json:"-"`
	UpdatedAt   time.Time `json:"-"`
}

// TableName 分类方案表表名
func (CategoryScheme) TableName() string {
	return "category_schemes"
}

// Category 分类信息 (categories 表). 视频分类的 Id 即影片的 cid/pid (分类ID全局唯一, 跨方案不重复);
// Pid 对应 parent_id, Show 对应 status (1 启用 / 0 停用), json 名保持原样供模板与前端使用
type Category struct {
	Id         int64                   `json:"id" gorm:"primaryKey"`
	SchemeId   int64                   `json:"schemeId" gorm:"column:scheme_id;default:1"` // 所属分类方案, 未设置时为默认方案
	Pid        int64                   `json:"pid" gorm:"column:parent_id"`
	Type       int                     `json:"type"`
	Name       string                  `json:"name"`
	Slug       string                  `json:"slug"` // 方案内唯一
	Show       bool                    `json:"show" gorm:"column:status"`
	Sort       int64                   `json:"sort"`                        // 同级排序, 越小越靠前
	I18n       map[string]CategoryText `json:"i18n" gorm:"serializer:json"` // 各语言的名称, 留空时使用原文
	SourceName string                  `json:"-" gorm:"-"`                  // Localize 前的原文名称 (主题按原文名称选分类图标)
	CreatedAt  time.Time               `json:"-"`
	UpdatedAt  time.Time               `json:"-"`
}

// CategoryText 分类在某个语言的文字 (分类页的 SEO 由方案的 SEO 规则决定)
type CategoryText struct {
	Name string `json:"name"`
}

// Localize 以 lang 的译名覆盖名称 (留空时保留原文), 原文名称保存在 SourceName
func (c *Category) Localize(lang string) {
	if c == nil {
		return
	}
	if c.SourceName == "" {
		c.SourceName = c.Name
	}
	t, ok := c.I18n[lang]
	if !ok {
		return
	}
	if t.Name != "" {
		c.Name = t.Name
	}
}

// mergeCategoryI18n 以请求中的语言覆盖原有译文, 请求中没有的语言 (如方案暂时停用的语言) 保留; 结果经 normalizeCategoryI18n
func mergeCategoryI18n(old, req map[string]CategoryText) map[string]CategoryText {
	if old == nil && req == nil {
		return nil
	}
	out := make(map[string]CategoryText, len(old)+len(req))
	for lang, t := range old {
		out[lang] = t
	}
	for lang, t := range req {
		out[lang] = t
	}
	return normalizeCategoryI18n(out)
}

// normalizeCategoryI18n 去除首尾空白, 删除译名为空的语言
func normalizeCategoryI18n(m map[string]CategoryText) map[string]CategoryText {
	if m == nil {
		return nil
	}
	out := map[string]CategoryText{}
	for lang, t := range m {
		t = CategoryText{Name: strings.TrimSpace(t.Name)}
		if t != (CategoryText{}) {
			out[lang] = t
		}
	}
	return out
}

// TableName 分类表表名
func (c *Category) TableName() string {
	return "categories"
}

// CategoryTree 分类信息树形结构 (由 categories 表按 parent_id 组装, 树根 Id 为 0)
type CategoryTree struct {
	*Category
	Children []*CategoryTree `json:"children"`
}

// Localize 递归本地化分类树
func (t *CategoryTree) Localize(lang string) {
	if t == nil {
		return
	}
	t.Category.Localize(lang)
	for _, c := range t.Children {
		c.Localize(lang)
	}
}
