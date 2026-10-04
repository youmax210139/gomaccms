package renderer

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"gomaccms/internal/view/maccms"
)

var tagRegexp = regexp.MustCompile(`<[^>]*>`)
var digitRegexp = regexp.MustCompile(`\d+`)
var whitespaceRegexp = regexp.MustCompile(`\s`)
var aliasRegexp = regexp.MustCompile(`(～.*～)`)
var contentCleanRegexp = regexp.MustCompile(`(&.*;)|(<[^>]+>)`)
var spaceRunRegexp = regexp.MustCompile(`[\s\x{3000}]+`)

// FuncMap holds the template functions available to every theme's pages.
var FuncMap = template.FuncMap{
	"formatRemarks":    formatRemarks,
	"formatYear":       formatYear,
	"formatArea":       formatArea,
	"add":              func(a, b int) int { return a + b },
	"categoryIcon":     categoryIcon,
	"limit":            limit,
	"stripAlias":       stripAlias,
	"cleanFilmContent": cleanFilmContent,
	"truncateList":     truncateList,
	"splitList":        splitList,
	"blank":            blank,
	"classTagMobile":   classTagMobile,
	"classifyURL":      classifyURL,
	"pageRange":        pageRange,
	"subtract":         func(a, b int) int { return a - b },
	"searchBlurb":      searchBlurb,
	"searchURL":        searchURL,
	"toJSON":           toJSON,
	"safeHTML":         safeHTML,
	"asset":            asset,
	"t":                translate,
	"tHTML":            translateHTML,
	"i18nTexts":        i18nTexts,
}

func init() {
	// MacCMS 标签翻译后使用的函数 (见 view/maccms)
	for k, v := range maccms.Funcs() {
		FuncMap[k] = v
	}
}

// stripHTML removes any HTML tags from s (film blurbs sometimes contain
// leftover markup from scraped sources).
func stripHTML(s string) string {
	return tagRegexp.ReplaceAllString(s, "")
}

// formatRemarks 把更新备注简化为「数字+单位」, 如 "更新至10集" -> "10集", "更新到5期" -> "5期";
// 没有数字或单位时原样返回
func formatRemarks(s string) string {
	digits := digitRegexp.FindString(s)
	if digits == "" {
		return s
	}
	switch {
	case containsRune(s, '期'):
		return digits + "期"
	case containsRune(s, '集'):
		return digits + "集"
	default:
		return s
	}
}

// formatYear 年份取前 4 个字符; 为空时返回 "" (模板用 or 显示语言包的「未知」)
func formatYear(s string) string {
	if whitespaceRegexp.ReplaceAllString(s, "") == "" {
		return ""
	}
	r := []rune(s)
	if len(r) > 4 {
		return string(r[:4])
	}
	return s
}

// formatArea 取第一个地区 (逗号分隔); 为空时返回 ""
func formatArea(s string) string {
	if whitespaceRegexp.ReplaceAllString(s, "") == "" {
		return ""
	}
	parts := strings.SplitN(s, ",", 2)
	return parts[0]
}

// categoryIcon picks an iconfont class for a top-level category name by
// substring match, checked in this exact order: "电影" (film) ->
// "icon-film"; else "剧" (drama/series) -> "icon-tv"; else "动漫" (anime)
// -> "icon-cartoon"; else "icon-variety". Mirrors Home.vue's inline
// ternary chain for the nav icon shown next to each section title.
func categoryIcon(name string) string {
	switch {
	case strings.Contains(name, "电影"):
		return "icon-film"
	case strings.Contains(name, "剧"):
		return "icon-tv"
	case strings.Contains(name, "动漫"):
		return "icon-cartoon"
	default:
		return "icon-variety"
	}
}

// limit returns at most the first n elements of list (any slice type),
// or list unchanged if it isn't a slice or already has n or fewer
// elements. Used to cap film-grid/hot-list rendering the same way
// Home.vue's slice(0,12) did, without needing a typed helper per slice
// element type.
func limit(n int, list interface{}) interface{} {
	if n < 0 {
		n = 0
	}
	v := reflect.ValueOf(list)
	if v.Kind() != reflect.Slice || v.Len() <= n {
		return list
	}
	return v.Slice(0, n).Interface()
}

func containsRune(s string, r rune) bool {
	for _, c := range s {
		if c == r {
			return true
		}
	}
	return false
}

// stripAlias removes a "～alias～" suffix from a film name, e.g.
// "浪客剑心～追忆篇～" -> "浪客剑心".
func stripAlias(s string) string {
	return aliasRegexp.ReplaceAllString(s, "")
}

// cleanFilmContent 清理简介中的 HTML 实体与标签并整理空白 (见 tidySpaces; 注意 "&.*;" 为贪婪匹配)
func cleanFilmContent(s string) string {
	return tidySpaces(contentCleanRegexp.ReplaceAllString(s, ""))
}

// wide 中日韩文字与全角标点 (这些字符旁边的空白不需要)
func wide(r rune) bool {
	return unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana, unicode.Hangul) ||
		(r >= 0x3000 && r <= 0x303F) || (r >= 0xFF00 && r <= 0xFFEF)
}

// tidySpaces 连续空白 (含全角空格、换行) 合并为一个空格, 中日韩文字旁的空格去掉; 拉丁文字 (越南文、英文) 的词间空格保留
func tidySpaces(s string) string {
	s = strings.TrimSpace(spaceRunRegexp.ReplaceAllString(s, " "))
	rs := []rune(s)
	var b strings.Builder
	for i, r := range rs {
		if r == ' ' && (wide(rs[i-1]) || wide(rs[i+1])) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// listSepRegexp 演员 / 导演等名单的分隔符
var listSepRegexp = regexp.MustCompile(`[,，、/|]+`)

// splitList splits a name list (actors, directors) on common separators,
// trims blanks, and keeps at most n entries (n <= 0 keeps all).
func splitList(s string, n int) []string {
	var out []string
	for _, p := range listSepRegexp.Split(s, -1) {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
			if n > 0 && len(out) >= n {
				break
			}
		}
	}
	return out
}

// truncateList 只保留逗号分隔名单的前 3 个, 以空格连接
func truncateList(s string) string {
	parts := strings.Split(s, ",")
	var b strings.Builder
	for i, p := range parts {
		if i >= 3 {
			break
		}
		b.WriteString(p)
		b.WriteString(" ")
	}
	return strings.TrimRight(b.String(), " ")
}

// blank s 是否为空或只有空白
func blank(s string) bool {
	return whitespaceRegexp.ReplaceAllString(s, "") == ""
}

// classTagMobile 剧情标签以 " | " 连接; 为空时返回 ""
func classTagMobile(s string) string {
	if s == "" {
		return ""
	}
	return strings.ReplaceAll(s, ",", " | ")
}

// classifyURL 筛选页链接: 沿用当前筛选条件, 改写 overrideKey (为空时不改, 用于分页链接) 并指定页码
func classifyURL(params map[string]string, overrideKey, overrideValue string, current int) string {
	q := url.Values{}
	for _, k := range []string{"Pid", "Category", "Plot", "Area", "Language", "Year", "Sort"} {
		v := params[k]
		if k == overrideKey {
			v = overrideValue
		}
		if v != "" {
			q.Set(k, v)
		}
	}
	q.Set("current", strconv.Itoa(current))
	return "/filmClassifySearch?" + q.Encode()
}

// pageRange 以 current 为中心、最多 count 个页码, 限制在 [1, total] 内
func pageRange(current, total, count int) []int {
	if total <= 0 || count <= 0 {
		return nil
	}
	half := count / 2
	start := current - half
	if start < 1 {
		start = 1
	}
	end := start + count - 1
	if end > total {
		end = total
		start = end - count + 1
		if start < 1 {
			start = 1
		}
	}
	r := make([]int, 0, end-start+1)
	for i := start; i <= end; i++ {
		r = append(r, i)
	}
	return r
}

// searchBlurb 搜索结果卡片的简介: 去掉 HTML 标签与所有空白; 为空时返回 ""
func searchBlurb(s string) string {
	stripped := stripHTML(s)
	if strings.TrimSpace(stripped) == "" {
		return ""
	}
	return tidySpaces(stripped)
}

// searchURL 搜索页链接 (关键字已转义)
func searchURL(keyword string, current int) string {
	return "/search?search=" + url.QueryEscape(keyword) + "&current=" + strconv.Itoa(current)
}

// toJSON 输出可嵌入 <script> 的 JSON (encoding/json 会转义 <, >, &, 内容无法跳出 script 标签)
func toJSON(v interface{}) template.JS {
	b, err := json.Marshal(v)
	if err != nil {
		return template.JS("null")
	}
	return template.JS(b)
}

// safeHTML outputs s without escaping. Only for trusted, admin-authored
// content such as the site's analytics code.
func safeHTML(s string) template.HTML {
	return template.HTML(s)
}

// asset 主题静态资源的地址, 附上文件修改时间 (?v=) 让浏览器在文件更新后重新下载;
// 用法 {{asset .ThemeName "css/site.css"}}
func asset(theme, path string) string {
	u := fmt.Sprintf("/static/%s/%s", theme, path)
	if fi, err := os.Stat(filepath.Join(ThemesRoot, theme, "public", path)); err == nil {
		u += "?v=" + strconv.FormatInt(fi.ModTime().Unix(), 36)
	}
	return u
}
