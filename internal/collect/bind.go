package collect

import (
	"gomaccms/internal/db"
	"log"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CategoryBind 采集接口分类绑定: 采集站分类 TypeId → 本站分类 CategoryId; 一个采集站分类可绑定多个本站分类 (可跨分类方案)
type CategoryBind struct {
	SourceId   string    `json:"sourceId" gorm:"primaryKey"`
	TypeId     int64     `json:"typeId" gorm:"primaryKey;autoIncrement:false"`
	TypeName   string    `json:"typeName"`
	CategoryId int64     `json:"categoryId" gorm:"primaryKey;autoIncrement:false"`
	CreatedAt  time.Time `json:"-"`
	UpdatedAt  time.Time `json:"-"`
}

// TableName 分类绑定表表名
func (b CategoryBind) TableName() string {
	return "collect_binds"
}

// GetBindMap 返回采集站的分类绑定 TypeId → 本站分类ID 列表
func (r *Repository) GetBindMap(sourceId string) map[int64][]int64 {
	var l []CategoryBind
	if err := db.Mdb.Where("source_id = ?", sourceId).Order("type_id, category_id").Find(&l).Error; err != nil {
		log.Println("Get collect binds failed:", err)
	}
	m := make(map[int64][]int64, len(l))
	for _, b := range l {
		m[b.TypeId] = append(m[b.TypeId], b.CategoryId)
	}
	return m
}

// SetBinds 覆盖采集站分类的绑定, categoryIds 为空时解除绑定
func (r *Repository) SetBinds(sourceId string, typeId int64, typeName string, categoryIds []int64) error {
	return db.Mdb.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("source_id = ? AND type_id = ?", sourceId, typeId).Delete(&CategoryBind{}).Error; err != nil {
			return err
		}
		var rows []CategoryBind
		seen := make(map[int64]bool)
		for _, id := range categoryIds {
			if id != 0 && !seen[id] {
				seen[id] = true
				rows = append(rows, CategoryBind{SourceId: sourceId, TypeId: typeId, TypeName: typeName, CategoryId: id})
			}
		}
		if len(rows) == 0 {
			return nil
		}
		return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&rows).Error
	})
}

// ClearBinds 清空指定采集站的分类绑定
func (r *Repository) ClearBinds(sourceIds ...string) error {
	if len(sourceIds) == 0 {
		return nil
	}
	return db.Mdb.Where("source_id IN ?", sourceIds).Delete(&CategoryBind{}).Error
}
