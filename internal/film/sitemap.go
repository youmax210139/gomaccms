package film

import (
	"gomaccms/internal/db"

	"gorm.io/gorm"
)

// VodsChanged 影片新增 / 修改 / 删除后调用 (由 bootstrap 注入 publish.Svc.VodsChanged, film 不依赖 publish):
// 清除缓存、标记 sitemap 需要重建、推送 IndexNow. schemes 为受影响的分类方案 (见 vodSchemes, 删除时需在删除前取得),
// 为 nil 表示全部方案 (如分类停用、转移); ids 为空表示影响范围不确定, 不推送 IndexNow
var VodsChanged = func(ids []int64, schemes []int64) {}

// vodSchemes 影片所在的分类方案 (不在任何分类中时为空切片, 不是 nil)
func vodSchemes(ids []int64) []int64 {
	schemes := []int64{}
	if len(ids) > 0 {
		db.Mdb.Model(&VodType{}).Where("vod_id IN ?", ids).Distinct("scheme_id").Pluck("scheme_id", &schemes)
	}
	return schemes
}

// SitemapVod sitemap 中的一部影片: ID 与内容更新时间
type SitemapVod struct {
	VodId   int64 `gorm:"column:vod_id"`
	VodTime int64 `gorm:"column:vod_time"`
}

// sitemapVodQuery 前台可见的影片: 已审核, 且在 typeIds 中至少一个未停用的分类里 (typeIds 为前台可看内容页的分类)
func sitemapVodQuery(tx *gorm.DB, typeIds []int64) *gorm.DB {
	return tx.Model(&Vod{}).Where("vod_status = ?", vodStatusOn).
		Where("EXISTS (SELECT 1 FROM vod_type t WHERE t.vod_id = vod.vod_id AND t.type_id IN ? AND t.status = ?)", typeIds, vodStatusOn)
}

// SitemapVodBatch 按 vod_id 递增逐批读取前台可见的影片 (只取 ID 与更新时间), afterId 为上一批最后的 ID
func SitemapVodBatch(typeIds []int64, afterId int64, limit int) ([]SitemapVod, error) {
	var l []SitemapVod
	if len(typeIds) == 0 {
		return l, nil
	}
	err := sitemapVodQuery(db.Mdb, typeIds).Select("vod_id", "vod_time").
		Where("vod_id > ?", afterId).Order("vod_id").Limit(limit).Find(&l).Error
	return l, err
}

// CountSitemapVods 前台可见的影片数
func CountSitemapVods(typeIds []int64) (int64, error) {
	var n int64
	if len(typeIds) == 0 {
		return 0, nil
	}
	err := sitemapVodQuery(db.Mdb, typeIds).Count(&n).Error
	return n, err
}

// LatestVods 最近更新的前台可见影片 (RSS 用)
func LatestVods(typeIds []int64, limit int) ([]Vod, error) {
	var l []Vod
	if len(typeIds) == 0 {
		return l, nil
	}
	err := sitemapVodQuery(db.Mdb, typeIds).Select(append(vodBasicColumns, "vod_content")).
		Order("vod_time DESC").Limit(limit).Find(&l).Error
	return l, err
}

// ResetSearchCache 让关键字搜索的结果缓存 (vod_search) 全部过期, 下次搜索重新查询; 保留搜索次数 (热搜用)
func ResetSearchCache() error {
	return db.Mdb.Model(&VodSearch{}).Where("1 = 1").Update("search_update_time", 0).Error
}
