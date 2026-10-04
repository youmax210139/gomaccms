package router

import (
	"gomaccms/internal/config"
	"gomaccms/internal/http/middleware"
	"gomaccms/internal/view/renderer"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {

	r := gin.Default()
	// 加载所有主题模板, 并将各主题的 public/ 目录注册为静态资源路由 (/static/<theme>/...)
	renderer.LoadThemes(r)
	// 开启跨域
	r.Use(middleware.Cors())

	// 上传图片静态资源 (storage/upload/gallery)
	r.Static(config.FilmPictureUrlPath, config.FilmPictureUploadDir)
	// admin (Inertia) 生产构建资源, gonertia viteAssets 预设以 /build/ 作为资源前缀
	r.Static("/build", "public/build")

	registerWebRoutes(r)
	registerAdminRoutes(r)

	return r
}
