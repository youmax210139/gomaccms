package seed

import (
	"encoding/json"
	"log"

	"gomaccms/internal/collect"
	"gomaccms/internal/config"
	"gomaccms/internal/db"
	"gomaccms/internal/util"

	"gorm.io/gorm/clause"
)

// seedCollectSources 将旧版存于 Redis 的采集站列表 (config.FilmSourceListKey) 导入 collect_sources 表:
// 仅在表为空时执行, 保留原站点ID (采集记录与附属播放源引用它); 旧的 syncPictures=true 转为「开启」.
// Redis 中的旧数据不删除; 没有旧数据时跳过, 由 bootstrap.FilmSourceInit 写入默认站点.
func seedCollectSources() error {
	if collect.Repo.ExistCollectSourceList() {
		return nil
	}
	members := db.Rdb.ZRange(db.Cxt, config.FilmSourceListKey, 0, -1).Val()
	var l []collect.FilmSource
	for _, m := range members {
		var legacy struct {
			collect.FilmSource
			SyncPictures bool `json:"syncPictures"`
		}
		if err := json.Unmarshal([]byte(m), &legacy); err != nil {
			log.Println("seed: skip legacy collect source:", err)
			continue
		}
		s := legacy.FilmSource
		if s.Id == "" {
			s.Id = util.GenerateSalt()
		}
		if legacy.SyncPictures {
			s.SyncImage = collect.SyncImageOn
		}
		l = append(l, s)
	}
	if len(l) == 0 {
		return nil
	}
	log.Printf("seed: importing %d collect sources from the legacy Redis list", len(l))
	return db.Mdb.Clauses(clause.OnConflict{DoNothing: true}).Create(&l).Error
}
