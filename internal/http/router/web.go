package router

import (
	"net/http"

	"gomaccms/internal/http/handler/public"
	"gomaccms/internal/http/middleware"

	"github.com/gin-gonic/gin"
)

// registerWebRoutes 注册前台 SSR 页面路由(按域名解析主题渲染)以及前台用到的少量 JSON 接口
func registerWebRoutes(r *gin.Engine) {
	r.GET("/", func(c *gin.Context) { c.Redirect(http.StatusFound, "/index") })
	r.GET(`/index`, middleware.ResolveTheme(), public.Index)
	r.GET(`/cache/del`, public.IndexCacheDel)
	r.GET(`/config/basic`, middleware.ResolveTheme(), public.SiteBasicConfigJSON)
	r.GET(`/navCategory`, middleware.ResolveTheme(), public.CategoriesInfo)
	r.GET(`/filmDetail`, middleware.ResolveTheme(), public.FilmDetail)
	r.GET(`/play`, middleware.ResolveTheme(), public.FilmPlayInfo)
	r.GET(`/searchFilm`, middleware.ResolveTheme(), public.SearchFilm)
	r.GET(`/search`, middleware.ResolveTheme(), public.Search)
	r.GET(`/filmClassify`, middleware.ResolveTheme(), public.FilmClassify)
	r.GET(`/filmClassifySearch`, middleware.ResolveTheme(), public.FilmTagSearch)
	r.GET(`/history`, middleware.ResolveTheme(), public.History)
	r.GET(`/today`, middleware.ResolveTheme(), public.Today)
	r.GET(`/rank`, middleware.ResolveTheme(), public.Rank)
	// SEO: sitemap 索引与分片 (分片经 NoRoute 提供)、robots
	r.GET(`/sitemap.xml`, middleware.ResolveTheme(), public.Sitemap)
	r.GET(`/robots.txt`, public.Robots)
	r.GET(`/rss.xml`, middleware.ResolveTheme(), public.RSS)
	r.NoRoute(middleware.ResolveTheme(), public.NotFound)
	//r.GET(`/filmCategory`, controller.FilmCategory) 弃用
}
