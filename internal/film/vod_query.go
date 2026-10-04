package film

import (
	"gomaccms/internal/db"
	"log"
	"strings"

	"gorm.io/gorm"
)

// VodQuery 模板数据标签 ({maccms:vod}) 的影片查询条件; 由 view/maccms 校验过的参数转换而来, 这里只按白名单拼条件
type VodQuery struct {
	SchemeId int64   // 只取在该分类方案中 (有未停用分类) 的影片
	Ids      []int64 // 指定影片
	TypeIds  []int64 // 分类 (一级分类含其下全部)
	Class    string  // 剧情标签 (包含)
	Area     string
	Lang     string
	Year     string // 年份或 2010-2020 区间
	Letter   string
	State    string
	OrderBy  string // time / time_add / hits / score / id, 其他值按 time
	Desc     bool
	Offset   int
	Limit    int
}

// vodQueryOrder 排序栏位白名单
var vodQueryOrder = map[string]string{"time": "vod_time", "time_add": "vod_time_add", "hits": "vod_hits", "score": "vod_score", "id": "vod_id"}

// vodQueryWhere VodQuery 的筛选条件 (不含排序与分页)
func vodQueryWhere(tx *gorm.DB, q VodQuery) *gorm.DB {
	s := tx.Model(&Vod{}).Where("vod_status = ?", vodStatusOn).
		Where("vod_id IN (?)", tx.Model(&VodType{}).Select("vod_id").Where("scheme_id = ? AND status = ?", q.SchemeId, vodStatusOn))
	if len(q.Ids) > 0 {
		s = s.Where("vod_id IN ?", q.Ids)
	}
	if len(q.TypeIds) > 0 {
		s = s.Where("vod_id IN (?)", tx.Model(&VodType{}).Select("vod_id").
			Where("(type_id IN (?) OR type_id_1 IN (?)) AND status = ?", q.TypeIds, q.TypeIds, vodStatusOn))
	}
	if q.Class != "" {
		s = s.Where("vod_class LIKE ?", "%"+escapeLike(q.Class)+"%")
	}
	for col, v := range map[string]string{"vod_area": q.Area, "vod_lang": q.Lang, "vod_letter": q.Letter, "vod_state": q.State} {
		if v != "" {
			s = s.Where(col+" = ?", v)
		}
	}
	if from, to, ok := strings.Cut(q.Year, "-"); ok {
		s = s.Where("vod_year BETWEEN ? AND ?", from, to)
	} else if q.Year != "" {
		s = s.Where("vod_year = ?", q.Year)
	}
	return s
}

// vodQueryScope 筛选 + 排序 + 分页
func vodQueryScope(tx *gorm.DB, q VodQuery) *gorm.DB {
	col, ok := vodQueryOrder[q.OrderBy]
	if !ok {
		col = "vod_time"
	}
	dir := " ASC"
	if q.Desc {
		dir = " DESC"
	}
	return vodQueryWhere(tx, q).Order(col + dir).Order("vod_id" + dir).Offset(q.Offset).Limit(q.Limit)
}

// QueryVods 按 VodQuery 查询影片 (列表栏位); withTotal 为 true 时同时返回符合条件的总数 (分页)
func QueryVods(q VodQuery, withTotal bool) ([]Vod, int64) {
	var total int64
	if withTotal {
		if err := vodQueryWhere(db.Mdb, q).Count(&total).Error; err != nil {
			log.Println("QueryVods count error:", err)
		}
	}
	var vl []Vod
	if err := vodQueryScope(db.Mdb, q).Select(vodBasicColumns).Find(&vl).Error; err != nil {
		log.Println("QueryVods error:", err)
		return nil, total
	}
	return vl, total
}
