package film

import (
	"gomaccms/internal/db"
	"gomaccms/internal/i18n"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// VodText 影片在某个语言的文字, 留空的字段使用原文
type VodText struct {
	Name     string `json:"name" gorm:"column:name"`
	Sub      string `json:"sub" gorm:"column:sub"`
	Content  string `json:"content" gorm:"column:content"`
	Actor    string `json:"actor" gorm:"column:actor"`
	Director string `json:"director" gorm:"column:director"`
	Writer   string `json:"writer" gorm:"column:writer"`
	Remarks  string `json:"remarks" gorm:"column:remarks"`
	Area     string `json:"area" gorm:"column:area"`
	LangText string `json:"langText" gorm:"column:lang_text"` // 影片的「语言」栏位 (如 国语)
	Class    string `json:"class" gorm:"column:class"`        // 剧情标签
}

// VodI18n 影片译文 (vod_i18n 表), 每部影片每个语言一行; Manual 为后台人工编辑
type VodI18n struct {
	VodId     int64  `gorm:"column:vod_id;primaryKey;autoIncrement:false"`
	Lang      string `gorm:"column:lang;primaryKey"`
	VodText   `gorm:"embedded"`
	Manual    bool `gorm:"column:is_manual"`
	UpdatedAt time.Time
}

// TableName 影片译文表表名
func (VodI18n) TableName() string {
	return "vod_i18n"
}

// pick 译文非空时覆盖
func pick(dst *string, v string) {
	if v != "" {
		*dst = v
	}
}

func (t VodText) applyBasic(b *MovieBasicInfo) {
	pick(&b.Name, t.Name)
	pick(&b.SubTitle, t.Sub)
	pick(&b.Blurb, t.Content)
	pick(&b.Actor, t.Actor)
	pick(&b.Director, t.Director)
	pick(&b.Remarks, t.Remarks)
	pick(&b.Area, t.Area)
}

func (t VodText) applySearch(s *SearchInfo) {
	pick(&s.Name, t.Name)
	pick(&s.SubTitle, t.Sub)
	pick(&s.ClassTag, t.Class)
	pick(&s.Area, t.Area)
	pick(&s.Language, t.LangText)
	pick(&s.Remarks, t.Remarks)
}

func (t VodText) applyDetail(d *MovieDetail) {
	pick(&d.Name, t.Name)
	pick(&d.SubTitle, t.Sub)
	pick(&d.Content, t.Content)
	pick(&d.Actor, t.Actor)
	pick(&d.Director, t.Director)
	pick(&d.Writer, t.Writer)
	pick(&d.Remarks, t.Remarks)
	pick(&d.Area, t.Area)
	pick(&d.Language, t.LangText)
	pick(&d.ClassTag, t.Class)
}

func (t VodText) applyVod(v *Vod) {
	pick(&v.VodName, t.Name)
	pick(&v.VodSub, t.Sub)
	pick(&v.VodContent, t.Content)
	pick(&v.VodActor, t.Actor)
	pick(&v.VodDirector, t.Director)
	pick(&v.VodWriter, t.Writer)
	pick(&v.VodRemarks, t.Remarks)
	pick(&v.VodArea, t.Area)
	pick(&v.VodLang, t.LangText)
	pick(&v.VodClass, t.Class)
}

// translated lang 需要覆盖 (非原文语言)
func translated(lang string) bool {
	return lang != "" && lang != i18n.SourceLang
}

// VodTexts 一批影片在 lang 的译文; 原文语言不查询
func VodTexts(ids []int64, lang string) map[int64]VodText {
	if !translated(lang) || len(ids) == 0 {
		return nil
	}
	var rows []VodI18n
	if err := db.Mdb.Where("vod_id IN ? AND lang = ?", ids, lang).Find(&rows).Error; err != nil {
		log.Println("Load VodI18n Error:", err)
		return nil
	}
	m := make(map[int64]VodText, len(rows))
	for _, r := range rows {
		m[r.VodId] = r.VodText
	}
	return m
}

// categoryNames 一批分类在 lang 的名称 (列表卡片上的分类名); 原文语言不查询
func categoryNames(cids []int64, lang string) map[int64]string {
	if !translated(lang) || len(cids) == 0 {
		return nil
	}
	var cl []Category
	if err := db.Mdb.Where("id IN ?", cids).Find(&cl).Error; err != nil {
		log.Println("Load Category Names Error:", err)
		return nil
	}
	m := make(map[int64]string, len(cl))
	for i := range cl {
		cl[i].Localize(lang)
		m[cl[i].Id] = cl[i].Name
	}
	return m
}

func localizeBasics(list []MovieBasicInfo, texts map[int64]VodText, names map[int64]string) {
	for i := range list {
		if t, ok := texts[list[i].Id]; ok {
			t.applyBasic(&list[i])
		}
		pick(&list[i].CName, names[list[i].Cid])
	}
}

func localizeSearchInfos(list []SearchInfo, texts map[int64]VodText, names map[int64]string) {
	for i := range list {
		if t, ok := texts[list[i].Mid]; ok {
			t.applySearch(&list[i])
		}
		pick(&list[i].CName, names[list[i].Cid])
	}
}

// LocalizeBasics 影片卡片按语言覆盖文字与分类名 (每个列表两次查询)
func LocalizeBasics(list []MovieBasicInfo, lang string) {
	if !translated(lang) || len(list) == 0 {
		return
	}
	ids, cids := make([]int64, len(list)), make([]int64, len(list))
	for i, b := range list {
		ids[i], cids[i] = b.Id, b.Cid
	}
	localizeBasics(list, VodTexts(ids, lang), categoryNames(cids, lang))
}

// LocalizeSearchInfos 检索信息 (排行榜等) 按语言覆盖文字与分类名
func LocalizeSearchInfos(list []SearchInfo, lang string) {
	if !translated(lang) || len(list) == 0 {
		return
	}
	ids, cids := make([]int64, len(list)), make([]int64, len(list))
	for i, s := range list {
		ids[i], cids[i] = s.Mid, s.Cid
	}
	localizeSearchInfos(list, VodTexts(ids, lang), categoryNames(cids, lang))
}

// LocalizeDetail 影片详情按语言覆盖文字 (分类名由调用方处理)
func LocalizeDetail(d *MovieDetail, lang string) {
	if t, ok := VodTexts([]int64{d.Mid}, lang)[d.Mid]; ok {
		t.applyDetail(d)
	}
}

// LocalizeVods vod 记录 (RSS) 按语言覆盖文字
func LocalizeVods(list []Vod, lang string) {
	if !translated(lang) || len(list) == 0 {
		return
	}
	ids := make([]int64, len(list))
	for i, v := range list {
		ids[i] = v.VodId
	}
	texts := VodTexts(ids, lang)
	for i := range list {
		if t, ok := texts[list[i].VodId]; ok {
			t.applyVod(&list[i])
		}
	}
}

// VodTranslations 影片全部语言的译文 (后台编辑页)
func VodTranslations(vodId int64) map[string]VodText {
	var rows []VodI18n
	db.Mdb.Where("vod_id = ?", vodId).Find(&rows)
	m := make(map[string]VodText, len(rows))
	for _, r := range rows {
		m[r.Lang] = r.VodText
	}
	return m
}

// trimmed 去除首尾空白, 短栏位截断到 255 字
func (t VodText) trimmed() VodText {
	return VodText{Name: clip(t.Name, 255), Sub: clip(t.Sub, 255), Content: strings.TrimSpace(t.Content),
		Actor: clip(t.Actor, 255), Director: clip(t.Director, 255), Writer: clip(t.Writer, 255),
		Remarks: clip(t.Remarks, 255), Area: clip(t.Area, 255), LangText: clip(t.LangText, 255), Class: clip(t.Class, 255)}
}

// splitVodTranslations 后台保存的译文: 内容有变的语言整行替换 (人工), 全部为空的语言删除, 没变的不动; 原文语言忽略.
// stored 为已存的译文 (nil 表示不比对)
func splitVodTranslations(vodId int64, texts map[string]VodText, stored map[string]VodI18n) (save []VodI18n, del []string) {
	for lang, t := range texts {
		if !translated(lang) {
			continue
		}
		t = t.trimmed()
		old, exists := stored[lang]
		if t == (VodText{}) {
			if exists || stored == nil {
				del = append(del, lang)
			}
			continue
		}
		// 内容没变的语言不重新保存, 保留原有的人工 / 自动标记与更新时间
		if exists && old.VodText == t {
			continue
		}
		save = append(save, VodI18n{VodId: vodId, Lang: lang, VodText: t, Manual: true})
	}
	return save, del
}

// saveVodTranslations 保存后台编辑的译文; texts 中没有的语言不变
func saveVodTranslations(tx *gorm.DB, vodId int64, texts map[string]VodText) error {
	var rows []VodI18n
	if err := tx.Where("vod_id = ?", vodId).Find(&rows).Error; err != nil {
		return err
	}
	stored := make(map[string]VodI18n, len(rows))
	for _, r := range rows {
		stored[r.Lang] = r
	}
	save, del := splitVodTranslations(vodId, texts, stored)
	if len(del) > 0 {
		if err := tx.Where("vod_id = ? AND lang IN ?", vodId, del).Delete(&VodI18n{}).Error; err != nil {
			return err
		}
	}
	if len(save) == 0 {
		return nil
	}
	return tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(&save).Error
}
