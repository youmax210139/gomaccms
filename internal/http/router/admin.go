package router

import (
	"gomaccms/internal/http/handler/admin"
	"gomaccms/internal/http/middleware"
	"gomaccms/internal/view/inertia"

	"github.com/gin-gonic/gin"
)

// registerAdminRoutes 注册后台登录/登出路由与 /manage/* 管理路由
// (Inertia 页面路由使用 AuthTokenInertia, 纯 JSON 路由使用 AuthToken)
func registerAdminRoutes(r *gin.Engine) {
	r.GET(`/login`, inertia.Middleware(), admin.ShowLogin)
	r.POST(`/login`, inertia.Middleware(), admin.Login)
	r.GET(`/login/captcha`, admin.LoginCaptcha)
	// /logout 现在只被走 cookie 认证的 Inertia 后台调用(ManageHeader.vue), 且
	// Login 已经不再签发 header token, 所以这里改用 AuthTokenInertia() 从 cookie
	// 读取身份, 而不是已经没有任何来源会填充的 auth-token 请求头。
	r.GET(`/logout`, inertia.Middleware(), middleware.AuthTokenInertia(), admin.Logout)
	r.POST(`/changePassword`, middleware.AuthToken(), admin.UserPasswordChange)

	// 管理员API路由组
	manageRoute := r.Group(`/manage`)
	manageRoute.Use(inertia.Middleware())
	{
		manageRoute.GET(`/index`, middleware.AuthTokenInertia(), admin.ManageIndex)

		// 系统相关
		sysConfig := manageRoute.Group(`/config`)
		sysConfig.Use(middleware.AuthTokenInertia())
		{
			sysConfig.GET(`/basic`, admin.SiteBasicConfig)
			sysConfig.POST(`/basic/update`, admin.UpdateSiteBasic)
			sysConfig.POST(`/default/update`, admin.UpdateDefaultSite)
			// 域名 → 主题映射 (与基本设置同一设置页)
			sysConfig.POST(`/domain/add`, admin.DomainAdd)
			sysConfig.POST(`/domain/update`, admin.DomainUpdate)
			sysConfig.GET(`/domain/del`, admin.DomainDel)
			sysConfig.POST(`/domain/state`, admin.DomainState)
			sysConfig.POST(`/seo/update`, admin.SiteSEO)
		}

		// 轮播相关
		adRoute := manageRoute.Group(`ad`)
		adRoute.Use(middleware.AuthTokenInertia())
		{
			adRoute.GET(`/list`, admin.AdList)
			adRoute.POST(`/save`, admin.AdSave)
			adRoute.POST(`/state`, admin.AdState)
			adRoute.GET(`/del`, admin.AdDel)
		}
		banner := manageRoute.Group(`banner`)
		banner.Use(middleware.AuthTokenInertia())
		{
			banner.GET(`/list`, admin.BannerList)
			banner.POST(`/save`, admin.BannerSave)
			banner.POST(`/state`, admin.BannerState)
			banner.GET(`/del`, admin.BannerDel)
			banner.GET(`/vods`, middleware.AuthToken(), admin.BannerVods)
		}
		// 网站地图 (每个分类方案的 sitemap / RSS / IndexNow): 页面为 Inertia, 其余为 JSON (后台任务, 前端轮询状态)
		manageRoute.GET(`/publish/center`, middleware.AuthTokenInertia(), admin.PublishCenter)
		pub := manageRoute.Group(`/publish`)
		pub.Use(middleware.AuthToken())
		{
			pub.GET(`/status`, admin.PublishStatus)
			pub.POST(`/sitemap`, admin.PublishSitemap)
			pub.POST(`/rss`, admin.PublishRSS)
			pub.POST(`/indexnow`, admin.PublishIndexNow)
			pub.GET(`/jobs/:id`, admin.PublishJob)
		}
		// 缓存管理
		manageRoute.GET(`/cache/list`, middleware.AuthTokenInertia(), admin.CacheCenter)
		manageRoute.POST(`/cache/refresh/:target`, middleware.AuthToken(), admin.CacheRefresh)
		// 前台语言
		lang := manageRoute.Group(`/lang`)
		lang.Use(middleware.AuthTokenInertia())
		{
			lang.GET(`/list`, admin.LanguageList)
			lang.POST(`/save`, admin.LanguageSave)
			lang.POST(`/state`, admin.LanguageState)
		}

		// 用户相关
		userRoute := manageRoute.Group(`/user`)
		userRoute.Use(middleware.AuthToken())
		{
			userRoute.GET(`/info`, admin.UserInfo)
		}

		// 管理员 / 会员组 / 会员
		adminRoute := manageRoute.Group(`/admin`)
		adminRoute.Use(middleware.AuthTokenInertia())
		{
			adminRoute.GET(`/list`, admin.AdminList)
			adminRoute.POST(`/save`, admin.AdminSave)
			adminRoute.POST(`/state`, admin.AdminState)
			adminRoute.POST(`/del`, admin.AdminDel)
			adminRoute.GET(`/log/list`, admin.AdminLogList)
			adminRoute.POST(`/log/clear`, admin.AdminLogClear)
		}
		// 个人资料 (所有管理员可用; /manage/user 分组是 JSON 路由, 页面路由单独注册)
		manageRoute.GET(`/user/profile`, middleware.AuthTokenInertia(), admin.Profile)
		manageRoute.POST(`/user/profile`, middleware.AuthTokenInertia(), admin.ProfileSave)
		manageRoute.POST(`/user/password`, middleware.AuthTokenInertia(), admin.ProfilePassword)
		memberRoute := manageRoute.Group(`/member`)
		memberRoute.Use(middleware.AuthTokenInertia())
		{
			memberRoute.GET(`/group/list`, admin.MemberGroupList)
			memberRoute.POST(`/group/save`, admin.MemberGroupSave)
			memberRoute.POST(`/group/state`, admin.MemberGroupState)
			memberRoute.POST(`/group/del`, admin.MemberGroupDel)
			memberRoute.GET(`/list`, admin.MemberList)
			memberRoute.POST(`/save`, admin.MemberSave)
			memberRoute.POST(`/state`, admin.MemberState)
			memberRoute.POST(`/del`, admin.MemberDel)
		}

		// 采集路相关
		collect := manageRoute.Group(`/collect`)
		collect.Use(middleware.AuthTokenInertia())
		{
			collect.GET(`/list`, admin.FilmSourceList)
			collect.GET(`/find`, middleware.AuthToken(), admin.FindFilmSource)
			collect.POST(`/test`, middleware.AuthToken(), admin.FilmSourceTest)
			collect.GET(`/add`, admin.FilmSourceAddPage)
			collect.POST(`/add`, admin.FilmSourceAdd)
			collect.GET(`/edit`, admin.FilmSourceEditPage)
			collect.POST(`/update`, admin.FilmSourceUpdate)
			collect.POST(`/change`, admin.FilmSourceChange)
			collect.GET(`/del`, admin.FilmSourceDel)
			collect.POST(`/del`, admin.FilmSourceDel)
			// 浏览采集站资源与分类绑定
			collect.GET(`/browse`, admin.CollectBrowse)
			collect.POST(`/browse/collect`, middleware.AuthToken(), admin.CollectFilmByIds)
			collect.GET(`/progress`, middleware.AuthToken(), admin.CollectProgress)
			collect.GET(`/logs`, middleware.AuthToken(), admin.CollectLogList)
			collect.GET(`/clear/preview`, middleware.AuthToken(), admin.CollectClearPreview)
			collect.POST(`/clear`, middleware.AuthToken(), admin.CollectClear)
			collect.POST(`/bind`, admin.CollectBind)
			collect.POST(`/bind/clear`, admin.CollectBindClear)
			collect.GET(`/options`, middleware.AuthToken(), admin.GetNormalFilmSource)
		}

		// 定时任务相关
		collectCron := manageRoute.Group(`/cron`)
		collectCron.Use(middleware.AuthTokenInertia())
		{
			collectCron.GET(`/list`, admin.FilmCronTaskList)
			collectCron.GET(`/find`, middleware.AuthToken(), admin.GetFilmCronTask)
			collectCron.POST(`/add`, admin.FilmCronAdd)
			collectCron.POST(`/update`, admin.FilmCronUpdate)
			collectCron.POST(`/change`, admin.ChangeTaskState)
			collectCron.GET(`/del`, admin.DelFilmCron)
		}
		// spider 数据采集
		spiderRoute := manageRoute.Group(`/spider`)
		spiderRoute.Use(middleware.AuthToken())
		{
			spiderRoute.POST(`/start`, admin.StarSpider)
			spiderRoute.POST(`/stop`, admin.StopSpider)
			spiderRoute.POST(`/resume`, admin.ResumeSpider)
			spiderRoute.POST(`/retry`, admin.RetryFailedPages)
			spiderRoute.GET(`/update/single`, admin.SingleUpdateSpider)
		}
		// filmManage 影视管理
		filmRoute := manageRoute.Group(`/film`)
		filmRoute.Use(middleware.AuthTokenInertia())
		{
			filmRoute.GET(`/add`, admin.FilmAddPage)
			filmRoute.POST(`/add`, admin.FilmAdd)
			filmRoute.GET(`/edit`, admin.FilmEditPage)
			filmRoute.POST(`/edit`, admin.FilmEdit)
			filmRoute.GET(`/search/list`, admin.FilmSearchPage)
			filmRoute.GET(`/search/del`, admin.FilmDelete)
			filmRoute.POST(`/search/batch/del`, admin.FilmBatchDelete)
			filmRoute.POST(`/search/batch/update`, admin.FilmBatchUpdate)
			// 播放器
			filmRoute.GET(`/player`, admin.PlayerList)
			filmRoute.POST(`/player/save`, admin.PlayerSave)
			filmRoute.POST(`/player/state`, admin.PlayerState)
			filmRoute.GET(`/player/del`, admin.PlayerDel)

			filmRoute.GET(`/class/tree`, admin.FilmClassTree)
			filmRoute.GET(`/class/find`, middleware.AuthToken(), admin.FindFilmClass)
			filmRoute.POST(`/class/save`, admin.SaveFilmClass)
			filmRoute.POST(`/class/update`, admin.UpdateFilmClass)
			filmRoute.GET(`/class/del`, admin.DelFilmClass)
			filmRoute.POST(`/class/batch/del`, admin.BatchFilmClassDel)
			filmRoute.POST(`/class/batch/state`, admin.BatchFilmClassState)
			filmRoute.POST(`/class/transfer`, admin.TransferFilmClass)
			filmRoute.POST(`/class/scheme/save`, admin.SaveCategoryScheme)
			filmRoute.POST(`/class/scheme/seo`, admin.SaveSEORules)
			filmRoute.GET(`/class/scheme/del`, admin.DelCategoryScheme)
		}

		// 文件管理
		fileRoute := manageRoute.Group(`/file`)
		fileRoute.Use(middleware.AuthTokenInertia())
		{
			fileRoute.GET(`/upload`, admin.FileGallery)
			fileRoute.POST(`/upload`, admin.SingleUpload)
			fileRoute.GET(`/upload/multiple`, admin.MultipleUpload)
			fileRoute.GET(`/del`, admin.DelFile)
			fileRoute.GET(`/list`, middleware.AuthToken(), admin.FileList)
			fileRoute.GET(`/gallery`, admin.FileGalleryRedirect)
		}
	}
}
