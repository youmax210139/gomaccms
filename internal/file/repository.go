package file

import (
	"encoding/json"
	"gomaccms/internal/config"
	"gomaccms/internal/db"
	"gomaccms/internal/paging"
	"log"

	"github.com/redis/go-redis/v9"
)

var Repo *Repository

type Repository struct{}

func NewRepository() *Repository {
	return &Repository{}
}

func (r *Repository) SaveGallery(f FileInfo) {
	db.Mdb.Create(&f)
}

func (r *Repository) ExistFileInfoByRid(rid int64) bool {
	// rid 为 0 表示没有关联影片 (手动上传的图片), 不能用来匹配影片封面
	if rid <= 0 {
		return false
	}
	var count int64
	db.Mdb.Model(&FileInfo{}).Where("relevance_id = ?", rid).Count(&count)
	return count > 0
}

func (r *Repository) GetFileInfoByRid(rid int64) FileInfo {
	var f FileInfo
	if rid <= 0 {
		return f
	}
	db.Mdb.Where("relevance_id = ?", rid).First(&f)
	return f
}

func (r *Repository) GetFileInfoById(id uint) FileInfo {
	var f = FileInfo{}
	db.Mdb.First(&f, id)
	return f
}

func (r *Repository) GetFileInfoPage(tl []string, source string, page *paging.Page) []FileInfo {
	var fl []FileInfo
	query := db.Mdb.Model(&FileInfo{}).Where("file_type IN ?", tl).Order("id DESC")
	switch source {
	case SourceUpload:
		query = query.Where("relevance_id IS NULL OR relevance_id = 0")
	case SourcePoster:
		query = query.Where("relevance_id > 0")
	}
	paging.Apply(query, page)
	if err := paging.Limit(query, page).Find(&fl).Error; err != nil {
		log.Println(err)
		return nil
	}
	return fl
}

func (r *Repository) DelFileInfo(id uint) {
	db.Mdb.Unscoped().Delete(&FileInfo{}, id)
}

func (r *Repository) SaveVirtualPic(pl []VirtualPicture) error {
	var zl []redis.Z
	for _, p := range pl {
		m, _ := json.Marshal(p)
		zl = append(zl, redis.Z{Score: float64(p.Id), Member: m})
	}
	return db.Rdb.ZAdd(db.Cxt, config.VirtualPictureKey, zl...).Err()
}

// PopVirtualPics 从待同步集合中弹出最多 count 条记录
func (r *Repository) PopVirtualPics(count int64) []VirtualPicture {
	sl := db.Rdb.ZPopMax(db.Cxt, config.VirtualPictureKey, count).Val()
	var pl []VirtualPicture
	for _, s := range sl {
		vp := VirtualPicture{}
		_ = json.Unmarshal([]byte(s.Member.(string)), &vp)
		pl = append(pl, vp)
	}
	return pl
}
