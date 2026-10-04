package middleware

import (
	"gomaccms/internal/film"
	"gomaccms/internal/i18n"
	"gomaccms/internal/theme"
	"strings"

	"github.com/gin-gonic/gin"
)

// ResolveTheme resolves the current request's Host header to its domains
// record via theme.Svc and stashes the theme name into the Gin context under
// "theme" (read by renderer.Render) and the record itself under "domain"
// (nil when the host isn't configured; read by public.RenderPage for the
// per-domain site info). 同时按域名的分类方案解析请求语言: "lang" 为使用的语言代码,
// "langs" 为前台可切换的语言 ([]i18n.Language)
func ResolveTheme() gin.HandlerFunc {
	return func(c *gin.Context) {
		host := c.Request.Host
		if i := strings.Index(host, ":"); i != -1 {
			host = host[:i]
		}
		d := theme.Svc.ResolveDomain(host)
		c.Set("theme", theme.ThemeOf(d))
		c.Set("domain", d)
		sc := film.CategorySvc.Scheme(d.SchemeOrDefault())
		cookie, _ := c.Cookie(i18n.CookieName)
		lang, langs := i18n.Svc.ForScheme(sc.DefaultLang, sc.Langs, cookie)
		c.Set("lang", lang)
		c.Set("langs", langs)
		c.Next()
	}
}
