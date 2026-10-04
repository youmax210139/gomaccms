package siteconfig

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"gomaccms/internal/db"

	"gorm.io/gorm"
)

// Banner 首页海报 (banners 表): 每个分类方案各自一组, 前台只显示启用的, 显示几个由主题模板决定;
// 海报名称可按语言设置 (I18n, 语言代码 → 文字), 没有设置的语言用主名称
type Banner struct {
	Id        int64                 `json:"id" gorm:"primaryKey"`
	SchemeId  int64                 `json:"schemeId"`
	Mid       int64                 `json:"mid"` // 绑定的影片 (须在该分类方案中), 为 0 时海报不链接
	Name      string                `json:"name"`
	Poster    string                `json:"poster"`
	Picture   string                `json:"picture"`
	Sort      int64                 `json:"sort"`
	Status    bool                  `json:"status"`
	I18n      map[string]BannerText `json:"i18n" gorm:"serializer:json"`
	CreatedAt time.Time             `json:"-"`
	UpdatedAt time.Time             `json:"-"`
}

// BannerText 海报在某个语言的文字
type BannerText struct {
	Name string `json:"name"`
}

// TableName 海报表表名
func (Banner) TableName() string {
	return "banners"
}

// Banners 一组海报
type Banners []Banner

// Text 指定语言的文字 (模板中 {{$t := $b.Text $.lang}}); 没有设置时用主名称
func (b Banner) Text(lang string) BannerText {
	t := BannerText{Name: b.Name}
	if tr := b.I18n[lang]; tr.Name != "" {
		t.Name = tr.Name
	}
	return t
}

// bannerQuery 分类方案的海报, 按排序值; onlyEnabled 为 true 时只取启用的 (前台)
func bannerQuery(tx *gorm.DB, schemeId int64, onlyEnabled bool) *gorm.DB {
	q := tx.Model(&Banner{}).Where("scheme_id = ?", schemeId)
	if onlyEnabled {
		q = q.Where("status = ?", true)
	}
	return q.Order("sort ASC, id ASC")
}

// Banners 分类方案的海报 (onlyEnabled 为 true 时只取启用的)
func (s *Service) Banners(schemeId int64, onlyEnabled bool) Banners {
	var l Banners
	if err := bannerQuery(db.Mdb, schemeId, onlyEnabled).Find(&l).Error; err != nil {
		return Banners{}
	}
	return l
}

// FindBanner 按 ID 查询海报
func (s *Service) FindBanner(id int64) (Banner, error) {
	var b Banner
	err := db.Mdb.First(&b, id).Error
	return b, err
}

// SaveBanner 新增 (Id 为 0) 或修改海报
func (s *Service) SaveBanner(b *Banner) error {
	b.Name = strings.TrimSpace(b.Name)
	b.Poster, b.Picture = strings.TrimSpace(b.Poster), strings.TrimSpace(b.Picture)
	switch {
	case b.Name == "" || utf8.RuneCountInString(b.Name) > 255:
		return errors.New("海报名称不能为空, 且不能超过 255 个字符")
	case b.Poster == "" && b.Picture == "":
		return errors.New("请设置海报或封面图片")
	case b.Sort < 0:
		return errors.New("排序值不能为负数")
	}
	for lang, t := range b.I18n {
		t = BannerText{Name: strings.TrimSpace(t.Name)}
		if t == (BannerText{}) {
			delete(b.I18n, lang)
		} else {
			b.I18n[lang] = t
		}
	}
	if b.Id == 0 {
		return db.Mdb.Create(b).Error
	}
	res := db.Mdb.Model(&Banner{}).Where("id = ?", b.Id).
		Select("mid", "name", "poster", "picture", "sort", "status", "i18n", "updated_at").Updates(b)
	if res.Error == nil && res.RowsAffected == 0 {
		return errors.New("海报不存在")
	}
	return res.Error
}

// SetBannerStatus 启用 / 停用海报
func (s *Service) SetBannerStatus(id int64, status bool) error {
	return db.Mdb.Model(&Banner{}).Where("id = ?", id).Update("status", status).Error
}

// DeleteBanner 删除海报
func (s *Service) DeleteBanner(id int64) error {
	return db.Mdb.Delete(&Banner{}, id).Error
}
