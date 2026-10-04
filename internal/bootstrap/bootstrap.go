// Package bootstrap 负责程序启动时的装配与一次性初始化工作:
// 装配各 feature 的 repository/service 单例, 执行数据库迁移与初始数据, 首次启动时写入默认配置,
// 以及恢复采集源与定时任务。
package bootstrap

import (
	"context"
	"gomaccms/internal/view/maccms/tags"
	"log"

	"gomaccms/database/migrations"
	"gomaccms/database/seed"
	"gomaccms/internal/collect"
	"gomaccms/internal/config"
	"gomaccms/internal/cron"
	"gomaccms/internal/file"
	"gomaccms/internal/film"
	"gomaccms/internal/i18n"
	"gomaccms/internal/index"
	"gomaccms/internal/member"
	"gomaccms/internal/publish"
	"gomaccms/internal/siteconfig"
	"gomaccms/internal/spider"
	"gomaccms/internal/theme"
	"gomaccms/internal/user"
)

// Wire 装配各 feature 的 package-level repository / service 单例,
// handler 与 middleware 直接调用这些 XxxSvc。必须在 DefaultDataInit 与路由注册之前调用。
func Wire() {
	user.Repo = user.NewRepository()
	user.Svc = user.NewService(user.Repo)
	member.Repo = member.NewRepository()
	member.Svc = member.NewService(member.Repo)
	siteconfig.Svc = siteconfig.NewService(siteconfig.NewRepository())
	i18n.Repo = i18n.NewRepository()
	i18n.Svc = i18n.NewService(i18n.Repo)

	collect.Repo = collect.NewRepository()
	collect.Svc = collect.NewService(collect.Repo)

	cron.Repo = cron.NewRepository()
	cron.Svc = cron.NewService(cron.Repo)
	cron.TaskRunner = spider.RunCronTask
	// 影片变动通知发布中心 (缓存失效、sitemap 标记、IndexNow); publish.Svc 在数据库就绪后建立
	film.VodsChanged = func(ids []int64, schemes []int64) {
		if publish.Svc != nil {
			publish.Svc.VodsChanged(ids, schemes)
		}
	}

	file.Repo = file.NewRepository()
	file.Svc = file.NewService(file.Repo)

	film.CategoryRepo = film.NewCategoryRepository()
	film.CategorySvc = film.NewCategoryService(film.CategoryRepo)

	film.MovieRepo = film.NewMovieRepository()
	film.SearchRepo = film.NewSearchRepository()
	film.Svc = film.NewService(film.MovieRepo, film.SearchRepo)

	index.Svc = &index.Service{}

	theme.Repo = theme.NewRepository()
	theme.Svc = theme.NewService(theme.Repo)
	// MacCMS 模板数据标签 ({maccms:vod} 等)
	tags.Register()
}

// Seed 初始数据 (每次启动与 cmd/migrate seed 共用, 幂等): MySQL 的 seed.Run (admin 账户等),
// 以及 Redis 中还没有站点配置时写入默认配置 (首次安装或清空 Redis 后)
func Seed() error {
	if err := seed.Run(); err != nil {
		return err
	}
	if !siteconfig.Svc.HasSiteBasic() {
		BasicConfigInit()
	}
	return nil
}

// DefaultDataInit 启动时的数据初始化: 执行 (或检查) MySQL 版本化迁移, 写入初始数据 (Seed), 并恢复采集相关配置
func DefaultDataInit() {
	ctx := context.Background()
	// MySQL 表结构: goose 版本化迁移 (database/migrations); MIGRATE_ON_START=false 时只检查
	if config.MigrateOnStart {
		if err := migrations.Up(ctx); err != nil {
			log.Panic(err)
		}
	} else if n, err := migrations.Pending(ctx); err != nil {
		log.Panic(err)
	} else if n > 0 {
		log.Panicf("有 %d 个数据库迁移尚未执行 (MIGRATE_ON_START=false), 请先执行: go run ./cmd/migrate migrate", n)
	}
	if err := Seed(); err != nil {
		log.Panic("seed: ", err)
	}
	// 补建缺少的分类检索标签 (Redis)
	film.SearchRepo.RebuildMissingTags()
	// 初始化影视来源列表信息, 并回复恢复定时任务
	SpiderInit()
	// 网站地图 (任务表需迁移完成后才能使用)
	publish.Svc = publish.NewService()
	publish.Svc.Start()
}
