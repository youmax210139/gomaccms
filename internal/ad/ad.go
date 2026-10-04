// Package ad 广告位管理: 每个分类方案各自一组图片广告 (图片 + 链接, 支持 gif 动图), 按广告位 (slot) 分组,
// 主题以 {maccms:ad slot="..."} 标签显示. 不支持任意 HTML / JS 广告代码
package ad

import (
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"gomaccms/internal/db"

	"gorm.io/gorm"
)

// Ad 广告 (ads 表)
type Ad struct {
	Id        int64     `json:"id" gorm:"primaryKey"`
	SchemeId  int64     `json:"schemeId"`
	Slot      string    `json:"slot"`  // 广告位代码, 如 play_top
	Name      string    `json:"name"`  // 后台辨识用, 也作图片的 alt
	Image     string    `json:"image"` // 图片地址 (站内 /upload/... 或 http(s)://)
	Link      string    `json:"link"`  // 点击后打开的地址, 为空时不链接
	Sort      int64     `json:"sort"`
	Status    bool      `json:"status"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

// TableName 广告表表名
func (Ad) TableName() string {
	return "ads"
}

// Slots 后台广告位的常用选项 (也可以自填); 默认主题使用 play_top / play_bottom
var Slots = map[string]string{
	"play_top": "播放页 · 播放器下方", "play_bottom": "播放页 · 选集下方",
	"detail_top": "详情页 · 上方", "index_top": "首页 · 上方", "list_top": "分类页 · 上方",
}

var (
	slotPattern = regexp.MustCompile(`^[a-z0-9_]{1,30}$`)
	// 图片与链接只接受站内路径 (单个 / 开头) 或 http(s) 网址
	urlPattern = regexp.MustCompile(`^(https?://[^\s"'<>]+|/[^/\s"'<>][^\s"'<>]*)$`)
)

// normalize 去除首尾空白并校验
func normalize(a *Ad) error {
	a.Slot, a.Name = strings.TrimSpace(a.Slot), strings.TrimSpace(a.Name)
	a.Image, a.Link = strings.TrimSpace(a.Image), strings.TrimSpace(a.Link)
	switch {
	case !slotPattern.MatchString(a.Slot):
		return errors.New("广告位只能是小写字母、数字与下划线 (如 play_top), 最多 30 个字")
	case a.Name == "" || utf8.RuneCountInString(a.Name) > 100:
		return errors.New("名称不能为空, 且不能超过 100 个字")
	case a.Image == "" || len(a.Image) > 1024 || !urlPattern.MatchString(a.Image):
		return errors.New("请设置图片 (站内地址或 http(s) 网址)")
	case len(a.Link) > 1024 || (a.Link != "" && !urlPattern.MatchString(a.Link)):
		return errors.New("链接只能是站内地址或 http(s) 网址")
	case a.Sort < 0:
		return errors.New("排序值不能为负数")
	}
	return nil
}

// slotQuery 分类方案某广告位中启用的广告 (前台), 按排序值
func slotQuery(tx *gorm.DB, schemeId int64, slot string) *gorm.DB {
	return tx.Model(&Ad{}).Where("scheme_id = ? AND slot = ? AND status = ?", schemeId, slot, true).Order("sort ASC, id ASC")
}

var Svc = &Service{}

// Service 广告的读取与保存
type Service struct{}

// Slot 分类方案某广告位中启用的广告 (前台), offset / limit 为分页
func (s *Service) Slot(schemeId int64, slot string, offset, limit int) ([]Ad, int64) {
	var total int64
	q := slotQuery(db.Mdb, schemeId, slot)
	q.Count(&total)
	var l []Ad
	q.Offset(offset).Limit(limit).Find(&l)
	return l, total
}

// List 分类方案的全部广告 (后台), 按广告位与排序值
func (s *Service) List(schemeId int64) []Ad {
	var l []Ad
	db.Mdb.Where("scheme_id = ?", schemeId).Order("slot ASC, sort ASC, id ASC").Find(&l)
	return l
}

// Find 按 ID 查询广告
func (s *Service) Find(id int64) (Ad, error) {
	var a Ad
	err := db.Mdb.First(&a, id).Error
	return a, err
}

// Save 新增 (Id 为 0) 或修改广告
func (s *Service) Save(a *Ad) error {
	if err := normalize(a); err != nil {
		return err
	}
	if a.Id == 0 {
		return db.Mdb.Create(a).Error
	}
	res := db.Mdb.Model(&Ad{}).Where("id = ?", a.Id).
		Select("slot", "name", "image", "link", "sort", "status", "updated_at").Updates(a)
	if res.Error == nil && res.RowsAffected == 0 {
		return errors.New("广告不存在")
	}
	return res.Error
}

// SetStatus 启用 / 停用广告
func (s *Service) SetStatus(id int64, status bool) error {
	return db.Mdb.Model(&Ad{}).Where("id = ?", id).Update("status", status).Error
}

// Delete 删除广告
func (s *Service) Delete(id int64) error {
	return db.Mdb.Delete(&Ad{}, id).Error
}

// DeleteScheme 删除分类方案的全部广告 (删除方案时)
func (s *Service) DeleteScheme(schemeId int64) error {
	return db.Mdb.Where("scheme_id = ?", schemeId).Delete(&Ad{}).Error
}
