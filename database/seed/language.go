package seed

import (
	"gomaccms/internal/db"
	"gomaccms/internal/i18n"
)

// seedLanguages 写入缺少的内置语言: 中文 (原文语言) 与越南文
func seedLanguages() error {
	for _, l := range []i18n.Language{
		{Code: i18n.SourceLang, Name: "中文", LibreCode: "zh", Enabled: true, Sort: 0},
		{Code: "vi", Name: "Tiếng Việt", LibreCode: "vi", Enabled: true, Sort: 1},
	} {
		if err := db.Mdb.Where("code = ?", l.Code).FirstOrCreate(&l).Error; err != nil {
			return err
		}
	}
	return nil
}
