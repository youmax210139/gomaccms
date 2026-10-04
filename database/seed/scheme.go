package seed

import (
	"gomaccms/internal/db"
	"gomaccms/internal/film"
	"gomaccms/internal/i18n"
	"gomaccms/internal/member"
)

// seedSchemes 写入默认分类方案 (ID 1, 不能删除; 未配置的域名与采集都使用它)
func seedSchemes() error {
	s := film.CategoryScheme{Id: film.DefaultSchemeId, Name: "默认方案", DefaultLang: i18n.SourceLang, Langs: []string{i18n.SourceLang}}
	return db.Mdb.Where("id = ?", s.Id).FirstOrCreate(&s).Error
}

// seedMemberGroups 写入内置会员组: 1 游客 (未登录访客) 与 2 默认会员 (不能删除、停用)
func seedMemberGroups() error {
	for _, g := range []member.Group{
		{Id: member.GuestGroupId, Name: "游客", Status: true},
		{Id: member.DefaultGroupId, Name: "默认会员", Status: true},
	} {
		if err := db.Mdb.Where("id = ?", g.Id).FirstOrCreate(&g).Error; err != nil {
			return err
		}
	}
	return nil
}
