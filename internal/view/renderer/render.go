package renderer

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

const defaultTheme = "default"

// Render executes page's template within the theme resolved onto c's
// context by middleware.ResolveTheme, injecting ThemeAssetPath/ThemeName
// into data, plus i18n (当前语言 data["lang"] 的语言包, 供 t / tHTML 使用).
func Render(c *gin.Context, page string, data map[string]any) {
	name, _ := c.Get("theme")
	themeName, _ := name.(string)
	if themeName == "" {
		themeName = defaultTheme
	}

	bundle, ok := Registry[themeName]
	if !ok {
		bundle, ok = Registry[defaultTheme]
		if !ok {
			log.Panicf("theme: no %q theme registered and no default theme fallback available", themeName)
		}
		themeName = defaultTheme
	}

	if reload {
		fresh, err := parseTheme(themeName)
		if err != nil {
			c.String(http.StatusInternalServerError, "%v", err)
			return
		}
		bundle = fresh
	}

	tmpl, ok := bundle.Pages[page]
	if !ok {
		c.String(http.StatusInternalServerError, "theme: page %q not found in theme %q", page, themeName)
		return
	}

	if data == nil {
		data = map[string]any{}
	}
	data["ThemeAssetPath"] = bundle.AssetPath
	data["ThemeName"] = themeName
	lang, _ := data["lang"].(string)
	if lang == "" {
		lang = sourceLang
		data["lang"] = lang
	}
	data["i18n"] = bundle.texts(lang)

	c.Writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(c.Writer, "layout", data); err != nil {
		log.Println("theme: render error:", err)
	}
}
