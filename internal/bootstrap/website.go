package bootstrap

import (
	"gomaccms/internal/siteconfig"
)

// BasicConfigInit 初始化网站基本配置信息
func BasicConfigInit() {
	var bc = siteconfig.BasicConfig{
		SiteInfo: siteconfig.SiteInfo{
			SiteName: "GoMacCMS",
			SeoTitle: "GoMacCMS - 在线观影",
			Keyword:  "在线视频, 免费观影",
			Describe: "自动采集, 多播放源集成,在线观影网站",
		},
		State:         true,
		Hint:          "网站升级中, 暂时无法访问 !!!",
		PageSize:      siteconfig.DefaultAdminPageSize,
		DuplicateRule: siteconfig.DefaultDuplicateRule,
	}
	_ = siteconfig.Svc.UpdateSiteBasic(bc)
}
