package film

import (
	"errors"
	"fmt"
	"gomaccms/internal/collect"
	"gomaccms/internal/db"
	"log"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var CategoryRepo *CategoryRepository

type CategoryRepository struct{}

func NewCategoryRepository() *CategoryRepository {
	return &CategoryRepository{}
}

// List 按 sort, id 顺序返回分类; typ 为 0 时返回全部类型, schemeId 为 0 时返回全部方案
func (r *CategoryRepository) List(typ int, schemeId int64) []Category {
	var cl []Category
	query := db.Mdb.Order("sort ASC, id ASC")
	if typ != 0 {
		query = query.Where("type = ?", typ)
	}
	if schemeId != 0 {
		query = query.Where("scheme_id = ?", schemeId)
	}
	if err := query.Find(&cl).Error; err != nil {
		log.Println("List Categories Error:", err)
	}
	return cl
}

// BuildTree 按 parent_id 将分类组装成树, 父分类不存在的分类挂在树根下
func BuildTree(cl []Category) CategoryTree {
	root := CategoryTree{Category: &Category{Id: 0, Pid: -1, Name: "分类信息", Show: true}}
	nodes := make(map[int64]*CategoryTree, len(cl))
	for i := range cl {
		nodes[cl[i].Id] = &CategoryTree{Category: &cl[i]}
	}
	for i := range cl {
		node := nodes[cl[i].Id]
		if parent, ok := nodes[cl[i].Pid]; ok && cl[i].Pid != 0 {
			parent.Children = append(parent.Children, node)
		} else {
			root.Children = append(root.Children, node)
		}
	}
	return root
}

// GetCategoryTree 默认方案的视频分类树 (采集、后台手动添加影片等使用)
func (r *CategoryRepository) GetCategoryTree() CategoryTree {
	return r.GetSchemeCategoryTree(DefaultSchemeId)
}

// GetSchemeCategoryTree 指定方案的视频分类树 (前台按请求域名的方案展示)
func (r *CategoryRepository) GetSchemeCategoryTree(schemeId int64) CategoryTree {
	return BuildTree(r.List(CategoryVideo, schemeId))
}

// ExistsCategoryTree 默认方案是否已有视频分类 (没有时采集主站会先采集分类)
func (r *CategoryRepository) ExistsCategoryTree() bool {
	var count int64
	if err := db.Mdb.Model(&Category{}).Where("type = ? AND scheme_id = ?", CategoryVideo, DefaultSchemeId).Count(&count).Error; err != nil {
		log.Println("ExistsCategoryTree Error", err)
	}
	return count > 0
}

// DefaultSlug 采集导入的分类使用的默认 slug, 可在后台修改
func DefaultSlug(id int64) string {
	return fmt.Sprintf("category-%d", id)
}

// GetChildrenTree 一级分类的子分类 (分类ID全局唯一, 与方案无关)
func (r *CategoryRepository) GetChildrenTree(id int64) []*CategoryTree {
	var cl []Category
	if err := db.Mdb.Where("parent_id = ? AND type = ?", id, CategoryVideo).Order("sort ASC, id ASC").Find(&cl).Error; err != nil {
		log.Println("GetChildrenTree Error:", err)
		return nil
	}
	children := make([]*CategoryTree, 0, len(cl))
	for i := range cl {
		children = append(children, &CategoryTree{Category: &cl[i]})
	}
	return children
}

// Find 按ID查找分类
func (r *CategoryRepository) Find(id int64) (*Category, error) {
	var c Category
	if err := db.Mdb.First(&c, id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// SlugTaken slug 是否已被同一方案中 exceptId 以外的分类使用
func (r *CategoryRepository) SlugTaken(slug string, exceptId, schemeId int64) bool {
	var count int64
	db.Mdb.Model(&Category{}).Where("slug = ? AND id <> ? AND scheme_id = ?", slug, exceptId, schemeId).Count(&count)
	return count > 0
}

// HasChildren 分类下是否有子分类
func (r *CategoryRepository) HasChildren(id int64) bool {
	var count int64
	db.Mdb.Model(&Category{}).Where("parent_id = ?", id).Count(&count)
	return count > 0
}

func (r *CategoryRepository) Create(c *Category) error {
	return db.Mdb.Create(c).Error
}

// Update 更新分类的全部可编辑字段 (所属方案创建后不变)
func (r *CategoryRepository) Update(c *Category) error {
	return db.Mdb.Model(&Category{}).Where("id = ?", c.Id).
		Select("updated_at", "type", "parent_id", "name", "slug", "status", "sort", "i18n").
		Updates(c).Error
}

// UpdateColumn 更新分类的单个字段 (状态 / 排序)
func (r *CategoryRepository) UpdateColumn(id int64, column string, value any) error {
	return db.Mdb.Model(&Category{}).Where("id = ?", id).Update(column, value).Error
}

// Delete 删除分类及其子分类, 连同影片与它们的关联、指向它们的采集绑定 (影片本身不删除)
func (r *CategoryRepository) Delete(id int64) error {
	return db.Mdb.Transaction(func(tx *gorm.DB) error {
		var ids []int64
		if err := tx.Model(&Category{}).Where("id = ? OR parent_id = ?", id, id).Pluck("id", &ids).Error; err != nil {
			return err
		}
		return deleteCategories(tx, ids)
	})
}

func deleteCategories(tx *gorm.DB, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	if err := tx.Where("type_id IN ?", ids).Delete(&VodType{}).Error; err != nil {
		return err
	}
	if err := tx.Where("category_id IN ?", ids).Delete(&collect.CategoryBind{}).Error; err != nil {
		return err
	}
	return tx.Where("id IN ?", ids).Delete(&Category{}).Error
}

// ------------------------------------------------ 分类方案 ------------------------------------------------

// ListSchemes 全部分类方案 (默认方案在前)
func (r *CategoryRepository) ListSchemes() []CategoryScheme {
	var sl []CategoryScheme
	if err := db.Mdb.Order(clause.Expr{SQL: "id = ? DESC, sort ASC, id ASC", Vars: []any{DefaultSchemeId}}).Find(&sl).Error; err != nil {
		log.Println("List CategorySchemes Error:", err)
	}
	return sl
}

// SchemeExists 分类方案是否存在
func (r *CategoryRepository) SchemeExists(id int64) bool {
	var count int64
	db.Mdb.Model(&CategoryScheme{}).Where("id = ?", id).Count(&count)
	return count > 0
}

// SaveScheme 新增 (Id 为 0) 或修改分类方案的名称、排序与语言设置
func (r *CategoryRepository) SaveScheme(s *CategoryScheme) error {
	if s.Id == 0 {
		return db.Mdb.Create(s).Error
	}
	res := db.Mdb.Model(&CategoryScheme{}).Where("id = ?", s.Id).Select("updated_at", "name", "sort", "default_lang", "langs").Updates(s)
	if res.Error == nil && res.RowsAffected == 0 {
		return errors.New("分类方案不存在")
	}
	return res.Error
}

// FindScheme 按 ID 查询分类方案
func (r *CategoryRepository) FindScheme(id int64) (CategoryScheme, error) {
	var s CategoryScheme
	err := db.Mdb.First(&s, id).Error
	return s, err
}

// DeleteScheme 删除分类方案及其全部分类 (调用方需先确认没有域名在使用)
func (r *CategoryRepository) DeleteScheme(id int64) error {
	return db.Mdb.Transaction(func(tx *gorm.DB) error {
		var ids []int64
		if err := tx.Model(&Category{}).Where("scheme_id = ?", id).Pluck("id", &ids).Error; err != nil {
			return err
		}
		if err := deleteCategories(tx, ids); err != nil {
			return err
		}
		return tx.Delete(&CategoryScheme{}, id).Error
	})
}

// CountBySchemes 各分类方案的分类数 (全部类型)
func (r *CategoryRepository) CountBySchemes() map[int64]int64 {
	var rows []struct {
		SchemeId int64
		N        int64
	}
	if err := db.Mdb.Model(&Category{}).Select("scheme_id, COUNT(*) AS n").Group("scheme_id").Scan(&rows).Error; err != nil {
		log.Println("CountBySchemes Error:", err)
	}
	m := make(map[int64]int64, len(rows))
	for _, r := range rows {
		m[r.SchemeId] = r.N
	}
	return m
}
