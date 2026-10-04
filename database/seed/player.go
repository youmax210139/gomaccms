package seed

import (
	"log"
	"strings"

	"gomaccms/internal/collect"
	"gomaccms/internal/config"
	"gomaccms/internal/db"
	"gomaccms/internal/film"
)

// seedPlayers 播放器表为空时, 导入旧的 Redis 播放组代码 → 采集接口 对应 (名称为采集接口名称),
// 以及影片中其余尚未设置的播放组代码 (名称为代码本身); 之后由采集自动新增
func seedPlayers() error {
	var count int64
	if err := db.Mdb.Model(&film.Player{}).Count(&count).Error; err != nil || count > 0 {
		return err
	}
	sources := make(map[string]string)
	for _, s := range collect.Repo.GetCollectSourceList() {
		sources[s.Id] = s.Name
	}
	for code, sourceId := range db.Rdb.HGetAll(db.Cxt, config.PlayFromSourceKey).Val() {
		if name := sources[sourceId]; name != "" {
			film.EnsurePlayers([]string{code}, name)
		}
	}
	var froms []string
	db.Mdb.Model(&film.Vod{}).Distinct("vod_play_from").Pluck("vod_play_from", &froms)
	for _, f := range froms {
		for _, code := range strings.Split(f, "$$$") {
			film.EnsurePlayers([]string{code}, code)
		}
	}
	db.Mdb.Model(&film.Player{}).Count(&count)
	if count > 0 {
		log.Printf("seed: %d players imported", count)
	}
	return nil
}
