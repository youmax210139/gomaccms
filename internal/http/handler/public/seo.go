package public

import (
	"errors"
	"net/http"
	"os"
	"strings"

	"gomaccms/internal/film"
	"gomaccms/internal/publish"
	"gomaccms/internal/siteconfig"

	"github.com/gin-gonic/gin"
)

// baseURL 请求的 协议://域名 (经 nginx 时取 X-Forwarded-Proto), 用于 sitemap、RSS、canonical 的绝对 URL
func baseURL(c *gin.Context) string {
	proto := "http"
	if c.Request.TLS != nil {
		proto = "https"
	}
	if p := c.GetHeader("X-Forwarded-Proto"); p == "http" || p == "https" {
		proto = p
	}
	return proto + "://" + c.Request.Host
}

// Sitemap /sitemap.xml: 请求域名所用分类方案的 sitemap 索引
func Sitemap(c *gin.Context) {
	serveSitemap(c, "sitemap.xml")
}

// serveSitemap 输出分类方案的 sitemap 文件; 尚未生成时 404
func serveSitemap(c *gin.Context, name string) {
	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.Status(http.StatusOK)
	err := publish.ServeSitemapFile(c.Writer, publish.SchemeDir(schemeId(c)), name, baseURL(c))
	if errors.Is(err, os.ErrNotExist) {
		c.Header("Content-Type", "text/plain; charset=utf-8")
		c.String(http.StatusNotFound, "sitemap not found")
	}
}

// Robots /robots.txt: 不收录后台、搜索与观看记录, 并指出 sitemap
func Robots(c *gin.Context) {
	c.String(http.StatusOK, "User-agent: *\nDisallow: /manage/\nDisallow: /login\nDisallow: /search\nDisallow: /history\n\nSitemap: %s/sitemap.xml\n", baseURL(c))
}

// NotFound 没有对应路由的请求: sitemap 分片 (/sitemap-vod-N.xml 等) 在这里提供, 其余 404
func NotFound(c *gin.Context) {
	name := strings.TrimPrefix(c.Request.URL.Path, "/")
	if c.Request.Method == http.MethodGet && strings.HasPrefix(name, "sitemap-") {
		serveSitemap(c, name)
		return
	}
	// IndexNow 验证文件 /<key>.txt
	if p, key := publish.Svc.IndexNow.KeyFile(); p != "" && c.Request.URL.Path == p {
		c.String(http.StatusOK, key)
		return
	}
	c.String(http.StatusNotFound, "404 page not found")
}

// RSS /rss.xml: 请求域名所用分类方案最近更新的影片 (频道名称与描述取自该域名的站点设置)
func RSS(c *gin.Context) {
	// RSS 按方案的默认语言 (与影片文字一致)
	site := localizeSite(withDomainSiteInfo(c, siteconfig.Svc.GetSiteBasicConfig()), film.CategorySvc.Scheme(schemeId(c)).DefaultLang)
	b, err := publish.Svc.RSS.Feed(schemeId(c), baseURL(c), publish.RSSChannel{Title: site.SiteName, Description: site.Describe})
	if err != nil {
		c.String(http.StatusInternalServerError, "rss unavailable")
		return
	}
	c.Data(http.StatusOK, "application/rss+xml; charset=utf-8", b)
}
