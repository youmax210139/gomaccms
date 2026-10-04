// Package seed writes the initial MySQL data a fresh GoMacCMS install needs.
// Every seeder is idempotent: it only inserts what is missing and never
// overwrites or deletes existing rows, so Run is safe on every startup.
// (Redis-side defaults such as site config, banners and cron tasks, plus the
// default collect sources, stay in internal/bootstrap; seedCategories and
// seedCollectSources read the legacy Redis data once to move it into MySQL.)
package seed

// Run executes all seeders in order.
func Run() error {
	// 默认分类方案与内置会员组: 其他数据 (分类、权限判断) 依赖它们, 先写入
	if err := seedSchemes(); err != nil {
		return err
	}
	if err := seedMemberGroups(); err != nil {
		return err
	}
	if err := seedAdmin(); err != nil {
		return err
	}
	if err := seedCategories(); err != nil {
		return err
	}
	if err := seedCollectSources(); err != nil {
		return err
	}
	if err := seedLanguages(); err != nil {
		return err
	}
	if err := seedBanners(); err != nil {
		return err
	}
	return seedPlayers()
}
