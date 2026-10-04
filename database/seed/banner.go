package seed

import (
	"log"

	"gomaccms/internal/db"
	"gomaccms/internal/film"
	"gomaccms/internal/siteconfig"
)

// defaultBanners 默认分类方案的预设首页海报 (不绑定影片, 可在后台「海报管理」修改)
var defaultBanners = []siteconfig.Banner{
	{Name: "肖申克的救贖", Poster: "https://image.tmdb.org/t/p/w500/9cqNxx0GxF0bflZmeSMuL5tnGzr.jpg",
		Picture: "https://image.tmdb.org/t/p/w1280/zfbjgQE1uSd9wiPTX4VzsLi0rGG.jpg"},
	{Name: "鬼滅之刃", Poster: "https://image.tmdb.org/t/p/w600_and_h900_bestv2/3exjjYTseefny9nYjSbkIblZZdK.jpg",
		Picture: "https://images8.alphacoders.com/118/thumb-1920-1180819.jpg"},
	{Name: "異能", Poster: "https://0.soompi.io/wp-content/uploads/2023/08/02051734/Ryu-Seung-Ryong-1.jpeg",
		Picture: "https://0.soompi.io/wp-content/uploads/2023/08/02051734/Ryu-Seung-Ryong-1.jpeg"},
}

// seedBanners 海报表为空时写入预设海报 (默认分类方案, 启用)
func seedBanners() error {
	var count int64
	if err := db.Mdb.Model(&siteconfig.Banner{}).Count(&count).Error; err != nil || count > 0 {
		return err
	}
	for _, b := range defaultBanners {
		b.SchemeId, b.Status = film.DefaultSchemeId, true
		if err := db.Mdb.Create(&b).Error; err != nil {
			return err
		}
	}
	log.Printf("seed: %d default banners", len(defaultBanners))
	return nil
}
