package i18n

import (
	"errors"
	"gomaccms/internal/db"
)

var Repo *Repository

type Repository struct{}

func NewRepository() *Repository {
	return &Repository{}
}

// List 全部语言 (排序值小的在前)
func (r *Repository) List() ([]Language, error) {
	var l []Language
	err := db.Mdb.Order("sort ASC, code ASC").Find(&l).Error
	return l, err
}

// Exists 语言代码是否已存在
func (r *Repository) Exists(code string) bool {
	var n int64
	db.Mdb.Model(&Language{}).Where("code = ?", code).Count(&n)
	return n > 0
}

// Create 新增语言
func (r *Repository) Create(l *Language) error {
	return db.Mdb.Create(l).Error
}

// Update 修改语言的名称、翻译服务代码、排序与状态 (代码不可修改)
func (r *Repository) Update(l *Language) error {
	res := db.Mdb.Model(&Language{}).Where("code = ?", l.Code).
		Select("updated_at", "name", "libre_code", "sort", "enabled").Updates(l)
	if res.Error == nil && res.RowsAffected == 0 {
		return errors.New("语言不存在")
	}
	return res.Error
}

// SetEnabled 启用 / 停用语言
func (r *Repository) SetEnabled(code string, enabled bool) error {
	res := db.Mdb.Model(&Language{}).Where("code = ?", code).Update("enabled", enabled)
	if res.Error == nil && res.RowsAffected == 0 {
		return errors.New("语言不存在")
	}
	return res.Error
}
