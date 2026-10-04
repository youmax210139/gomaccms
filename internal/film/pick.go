package film

import (
	"gomaccms/internal/db"
	"gomaccms/internal/paging"
	"log"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

// escapeLike 转义 LIKE 的通配符
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// schemeVodQuery 分类方案中 (有未停用分类) 的影片; keyword 比对片名, 纯数字时也比对影片ID
func schemeVodQuery(tx *gorm.DB, schemeId int64, keyword string) *gorm.DB {
	q := tx.Model(&Vod{}).Where("vod_id IN (?)",
		tx.Model(&VodType{}).Select("vod_id").Where("scheme_id = ? AND status = ?", schemeId, vodStatusOn))
	kw := strings.TrimSpace(keyword)
	if kw == "" {
		return q
	}
	like := "%" + escapeLike(kw) + "%"
	if id, err := strconv.ParseInt(kw, 10, 64); err == nil {
		return q.Where("(vod_id = ? OR vod_name LIKE ?)", id, like)
	}
	return q.Where("vod_name LIKE ?", like)
}

// SchemeVods 后台挑选影片 (海报绑定等): 分类方案中的影片, 最近更新的在前
func SchemeVods(schemeId int64, keyword string, page *paging.Page) []MovieBasicInfo {
	q := schemeVodQuery(db.Mdb, schemeId, keyword)
	paging.Apply(q, page)
	var vl []Vod
	if err := paging.Limit(q.Select(vodBasicColumns).Order("vod_time DESC"), page).Find(&vl).Error; err != nil {
		log.Println("SchemeVods Error:", err)
		return nil
	}
	list := make([]MovieBasicInfo, 0, len(vl))
	for _, v := range vl {
		list = append(list, v.Basic())
	}
	return list
}
