package renderer

import (
	"encoding/json"
	"fmt"
	"html/template"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// sourceLang 原文语言, 与 i18n.SourceLang 相同 (renderer 不依赖 i18n 包); 其他语言包缺少的文字回退到它
const sourceLang = "zh-CN"

// loadTexts 读取主题的语言包 themes/<name>/lang/<code>.json (key → 文字), 每个语言都合并在原文语言包之上;
// 没有 lang 目录时返回空
func loadTexts(dir string) (map[string]map[string]string, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}
	raw := map[string]map[string]string{}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		var m map[string]string
		if err := json.Unmarshal(b, &m); err != nil {
			return nil, fmt.Errorf("theme lang %s: %w", f, err)
		}
		raw[strings.TrimSuffix(filepath.Base(f), ".json")] = m
	}
	out := make(map[string]map[string]string, len(raw))
	for lang, m := range raw {
		merged := make(map[string]string, len(raw[sourceLang])+len(m))
		maps.Copy(merged, raw[sourceLang])
		maps.Copy(merged, m)
		out[lang] = merged
	}
	return out, nil
}

// texts 指定语言的文字, 主题没有该语言包时用原文语言包
func (b *ThemeBundle) texts(lang string) map[string]string {
	if t, ok := b.Texts[lang]; ok {
		return t
	}
	return b.Texts[sourceLang]
}

// translate 模板函数 t: {{t $ "nav.today"}}、{{t $ "classify.home" .title.Name}}; 有参数时按 fmt 格式化, 没有该 key 时返回 key
func translate(data map[string]any, key string, args ...any) string {
	texts, _ := data["i18n"].(map[string]string)
	s, ok := texts[key]
	if !ok {
		s = key
	}
	if len(args) > 0 {
		s = fmt.Sprintf(s, args...)
	}
	return s
}

// translateHTML 模板函数 tHTML: 语言包文字含 HTML 标签时使用, 字符串参数先转义
func translateHTML(data map[string]any, key string, args ...any) template.HTML {
	esc := make([]any, len(args))
	for i, a := range args {
		if s, ok := a.(string); ok {
			esc[i] = template.HTMLEscapeString(s)
		} else {
			esc[i] = a
		}
	}
	return template.HTML(translate(data, key, esc...))
}

// i18nTexts 模板函数: 当前语言的全部文字, layout 中输出为 window.I18N 供主题 JS 使用
func i18nTexts(data map[string]any) map[string]string {
	if texts, ok := data["i18n"].(map[string]string); ok {
		return texts
	}
	return map[string]string{}
}

// T 供 handler 使用的翻译 (如页面 SEO 标题): 主题与语言取自 middleware.ResolveTheme 放进上下文的 theme / lang
func T(c *gin.Context, key string, args ...any) string {
	name := c.GetString("theme")
	bundle, ok := Registry[name]
	if !ok {
		bundle = Registry[defaultTheme]
	}
	var texts map[string]string
	if bundle != nil {
		texts = bundle.texts(c.GetString("lang"))
	}
	return translate(map[string]any{"i18n": texts}, key, args...)
}
