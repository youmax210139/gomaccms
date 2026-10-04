// Package seo 前台页面的 SEO (title / keywords / description): 每个分类方案按 页面类型 + 语言 设置规则,
// 规则是带 {占位符} 的文字, 由 Resolve 替换. 页面 SEO 的来源依次为 规则 → 站点 SEO (每个栏位各自回退)
package seo

import (
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// 页面类型
const (
	PageHome   = "home"   // 首页
	PageType   = "type"   // 分类页
	PageShow   = "show"   // 影片库筛选
	PageDetail = "detail" // 详情
	PagePlay   = "play"   // 播放
	PageSearch = "search" // 搜索
)

// PageTypes 支持的页面类型 (后台按此顺序显示)
var PageTypes = []string{PageHome, PageType, PageShow, PageDetail, PagePlay, PageSearch}

// Text 一组 title / keywords / description
type Text struct {
	Title       string `json:"title"`
	Keywords    string `json:"keywords"`
	Description string `json:"description"`
}

// Site 站点 (已按请求语言)
type Site struct {
	Name, URL, Keywords, Description string
}

// Type 当前分类 (已按请求语言)
type Type struct {
	Name string
}

// Vod 当前影片 (已按请求语言)
type Vod struct {
	Name, Sub, Year, Area, Lang, Class, Actor, Director, Remarks, Score, Content string
}

// Episode 当前播放的集
type Episode struct {
	Name  string
	Index int // 第几集 (1 起)
}

// Context 解析占位符所需的页面信息; 没有的部分为 nil
type Context struct {
	PageType string
	Site     Site
	Page     int
	Type     *Type
	Vod      *Vod
	Episode  *Episode
}

// defaultTitles 没有设置规则时的标题 (只用占位符, 任何语言都适用)
var defaultTitles = map[string]string{
	PageType:   "{type_name} - {site_name}",
	PageShow:   "{type_name} - {site_name}",
	PageDetail: "{vod_name} - {site_name}",
	PagePlay:   "{vod_name} {episode_name} - {site_name}",
}

// DefaultTitle 页面类型没有设置标题规则时使用的标题 (后台作为提示显示)
func DefaultTitle(pageType string) string {
	return defaultTitles[pageType]
}

var (
	placeholderPattern = regexp.MustCompile(`\{([a-z_]+)\}`)
	htmlTagPattern     = regexp.MustCompile(`<[^>]*>`)
	// repeatedComma 占位符为空时留下的连续逗号 (含全角)
	repeatedComma = regexp.MustCompile(`([,，])(\s*[,，])+`)
	// emptyParens 占位符为空时留下的空括号, 如 ({vod_year})
	emptyParens = regexp.MustCompile(`\(\s*\)|（\s*）`)
)

func currentYear() string {
	return strconv.Itoa(time.Now().Year())
}

// Values 占位符的值 (也供模板的 $obj / $param 使用); 没有的部分不在 map 中
func (c Context) Values() map[string]string {
	page := max(c.Page, 1)
	v := map[string]string{
		"site_name": c.Site.Name, "site_url": c.Site.URL, "site_keywords": c.Site.Keywords,
		"site_description": c.Site.Description, "page": strconv.Itoa(page), "year": currentYear(),
	}
	if t := c.Type; t != nil {
		v["type_name"] = t.Name
	}
	if d := c.Vod; d != nil {
		v["vod_name"], v["vod_sub"], v["vod_year"], v["vod_area"], v["vod_lang"] = d.Name, d.Sub, d.Year, d.Area, d.Lang
		v["vod_class"], v["vod_actor"], v["vod_director"], v["vod_remarks"], v["vod_score"] = d.Class, d.Actor, d.Director, d.Remarks, d.Score
		v["vod_content"] = summary(d.Content, 150)
	}
	if e := c.Episode; e != nil {
		v["episode_name"], v["episode_index"] = e.Name, strconv.Itoa(e.Index)
	}
	return v
}

// summary 去掉 HTML 与多余空白, 最多 n 个字
func summary(s string, n int) string {
	s = strings.Join(strings.Fields(htmlTagPattern.ReplaceAllString(s, " ")), " ")
	if utf8.RuneCountInString(s) > n {
		s = string([]rune(s)[:n])
	}
	return s
}

// Resolve 把规则中的 {占位符} 换成 ctx 中的值; 不认识或没有值的占位符换成空字串, 结果去掉多余空白
// 以及因此多出的逗号 (关键字).
// 输出是纯文字, 由 html/template 在 <title> / meta 中转义
func Resolve(tpl string, ctx Context) string {
	if tpl == "" {
		return ""
	}
	values := ctx.Values()
	out := placeholderPattern.ReplaceAllStringFunc(tpl, func(m string) string {
		return values[m[1:len(m)-1]]
	})
	out = emptyParens.ReplaceAllString(out, "")
	out = strings.Join(strings.Fields(out), " ")
	out = repeatedComma.ReplaceAllString(out, "$1")
	return strings.Trim(out, " ,，")
}

// Build 页面的 SEO: 每个栏位依次取 规则 (rule, 替换占位符; 标题没有规则时用默认规则) → 站点 SEO (site)
func Build(ctx Context, rule, site Text) Text {
	if rule.Title == "" {
		rule.Title = defaultTitles[ctx.PageType]
	}
	pick := func(rule, site string) string {
		if r := Resolve(rule, ctx); r != "" {
			return r
		}
		return site
	}
	return Text{Title: pick(rule.Title, site.Title), Keywords: pick(rule.Keywords, site.Keywords), Description: pick(rule.Description, site.Description)}
}
