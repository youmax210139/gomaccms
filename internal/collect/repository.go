package collect

import (
	"errors"
	"gomaccms/internal/db"
	"gomaccms/internal/util"
	"log"
)

var Repo *Repository

type Repository struct{}

func NewRepository() *Repository {
	return &Repository{}
}

func (r *Repository) SaveCollectSourceList(list []FilmSource) error {
	return db.Mdb.Create(&list).Error
}

func (r *Repository) GetCollectSourceList() []FilmSource {
	var l []FilmSource
	if err := db.Mdb.Order("created_at").Find(&l).Error; err != nil {
		log.Println(err)
		return nil
	}
	return l
}

func (r *Repository) FindCollectSourceById(id string) *FilmSource {
	var s FilmSource
	if err := db.Mdb.Where("id = ?", id).First(&s).Error; err != nil {
		return nil
	}
	return &s
}

func (r *Repository) DelCollectResource(id string) {
	if err := db.Mdb.Where("id = ?", id).Delete(&FilmSource{}).Error; err != nil {
		log.Println("Delete collect source failed:", err)
	}
}

// existSameApi 是否已有其他站点使用相同的接口地址+附加参数
func existSameApi(s FilmSource) bool {
	var n int64
	db.Mdb.Model(&FilmSource{}).Where("uri = ? AND params = ? AND id <> ?", s.Uri, s.Params, s.Id).Count(&n)
	return n > 0
}

func (r *Repository) AddCollectSource(s FilmSource) error {
	if existSameApi(s) {
		return errors.New("当前采集站点信息已存在, 请勿重复添加")
	}
	s.Id = util.GenerateSalt()
	return db.Mdb.Create(&s).Error
}

func (r *Repository) UpdateCollectSource(s FilmSource) error {
	if existSameApi(s) {
		return errors.New("当前采集站链接已存在其他站点中, 请勿重复添加")
	}
	return db.Mdb.Model(&FilmSource{}).Where("id = ?", s.Id).Select("*").Omit("id", "created_at").Updates(&s).Error
}

// UpdateCollectSourceState 只更新采集接口的启用状态
func (r *Repository) UpdateCollectSourceState(id string, state bool) error {
	return db.Mdb.Model(&FilmSource{}).Where("id = ?", id).Update("state", state).Error
}

func (r *Repository) ExistCollectSourceList() bool {
	var n int64
	db.Mdb.Model(&FilmSource{}).Count(&n)
	return n > 0
}
