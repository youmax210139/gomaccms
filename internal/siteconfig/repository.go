package siteconfig

import (
	"encoding/json"
	"gomaccms/internal/config"
	"gomaccms/internal/db"
	"log"
)

type Repository struct{}

func NewRepository() *Repository {
	return &Repository{}
}

func (r *Repository) GetSiteBasic() BasicConfig {
	c := BasicConfig{}
	data := db.Rdb.Get(db.Cxt, config.SiteConfigBasic).Val()
	if err := json.Unmarshal([]byte(data), &c); err != nil {
		log.Println("GetSiteBasic Err", err)
	}
	return c
}

// HasSiteBasic Redis 中是否已有站点配置
func (r *Repository) HasSiteBasic() bool {
	n, err := db.Rdb.Exists(db.Cxt, config.SiteConfigBasic).Result()
	return err == nil && n > 0
}

func (r *Repository) SaveSiteBasic(c BasicConfig) error {
	data, _ := json.Marshal(c)
	return db.Rdb.Set(db.Cxt, config.SiteConfigBasic, data, config.ManageConfigExpired).Err()
}
