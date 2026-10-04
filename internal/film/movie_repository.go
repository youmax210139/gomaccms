package film

import (
	"gomaccms/internal/db"
	"gomaccms/internal/file"
	"gomaccms/internal/paging"
	"gomaccms/internal/util"
	"log"
	"strings"

	"gorm.io/gorm/clause"
)

var MovieRepo *MovieRepository

type MovieRepository struct{}

func NewMovieRepository() *MovieRepository {
	return &MovieRepository{}
}

// SaveDetail 保存后台手动添加的影片 (Mid 为 0 时新增)
func (r *MovieRepository) SaveDetail(m MovieDetail) error {
	return UpsertVods([]MovieDetail{m})
}

// GetDetailByMid 影片详情 (mid 即 vod_id)
func (r *MovieRepository) GetDetailByMid(mid int64) MovieDetail {
	var vod Vod
	if res := db.Mdb.Where("vod_id = ?", mid).Limit(1).Find(&vod); res.Error != nil || res.RowsAffected == 0 {
		return MovieDetail{}
	}
	m := vod.Detail()
	replaceDetailPic(&m)
	return m
}

// GetBasicInfoByIds 按影片ID批量获取基本信息, 保持 ids 的顺序
func (r *MovieRepository) GetBasicInfoByIds(ids []int64) []MovieBasicInfo {
	if len(ids) == 0 {
		return nil
	}
	var vl []Vod
	if err := db.Mdb.Select(append(vodBasicColumns, "vod_content")).Where("vod_id IN ?", ids).Find(&vl).Error; err != nil {
		log.Println("BatchFind BasicInfo Failed: ", err)
	}
	found := make(map[int64]MovieBasicInfo, len(vl))
	for _, v := range vl {
		m := v.Detail()
		replaceDetailPic(&m)
		found[v.VodId] = ConvertBasicInfo(m)
	}
	l := make([]MovieBasicInfo, 0, len(found))
	for _, id := range ids {
		if b, ok := found[id]; ok {
			l = append(l, b)
		}
	}
	return l
}

// GetMovieListByPid 一级分类下的影片分页数据 (按更新时间)
func (r *MovieRepository) GetMovieListByPid(pid int64, page *paging.Page) []MovieBasicInfo {
	return r.getMovieList("type_id_1", pid, page)
}

// GetMovieListByCid 分类下的影片分页数据 (按更新时间)
func (r *MovieRepository) GetMovieListByCid(cid int64, page *paging.Page) []MovieBasicInfo {
	return r.getMovieList("type_id", cid, page)
}

func (r *MovieRepository) getMovieList(column string, id int64, page *paging.Page) []MovieBasicInfo {
	query := vodInCategory(column, id)
	paging.Apply(query, page)
	var ids []int64
	if err := paging.Limit(query.Order("vod_time DESC"), page).Pluck("vod_id", &ids).Error; err != nil {
		log.Println(err)
		return nil
	}
	return r.GetBasicInfoByIds(ids)
}

// GetRelateMovieBasicInfo 相关影片: 同分类下片名相近或剧情标签相同的影片, 片名相近的优先
func (r *MovieRepository) GetRelateMovieBasicInfo(search SearchInfo, page *paging.Page) []MovieBasicInfo {
	name := util.CleanFilmName(search.Name)
	tags := strings.ReplaceAll(util.FormatSpecialChar(strings.ReplaceAll(search.ClassTag, " ", "")), ",", " ")
	query := vodInCategory("type_id", search.Cid)
	switch {
	case name != "" && tags != "":
		query = query.Where("(MATCH(vod_name, vod_sub) AGAINST(?) OR MATCH(vod_class) AGAINST(?))", name, tags)
	case name != "":
		query = query.Where("MATCH(vod_name, vod_sub) AGAINST(?)", name)
	case tags != "":
		query = query.Where("MATCH(vod_class) AGAINST(?)", tags)
	}
	if name != "" {
		query = query.Order(clause.Expr{SQL: "MATCH(vod_name, vod_sub) AGAINST(?) DESC", Vars: []any{name}})
	}
	var ids []int64
	if err := paging.Limit(query.Order("vod_time DESC"), page).Pluck("vod_id", &ids).Error; err != nil {
		log.Println("GetRelateMovie Error:", err)
		return nil
	}
	return r.GetBasicInfoByIds(ids)
}

// replaceDetailPic 将影片详情中的图片地址替换为本地图片(如果已同步)
func replaceDetailPic(d *MovieDetail) {
	// 同步的封面以影片ID (vod_id) 关联
	if file.Repo.ExistFileInfoByRid(d.Mid) {
		d.Picture = file.Repo.GetFileInfoByRid(d.Mid).Link
	}
}
