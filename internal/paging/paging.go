// Package paging 提供各 feature 共用的分页参数结构与 GORM 分页计数辅助函数。
package paging

import "gorm.io/gorm"

// Page 分页信息结构体
type Page struct {
	PageSize  int `json:"pageSize"`
	Current   int `json:"current"`
	PageCount int `json:"pageCount"`
	Total     int `json:"total"`
}

// PagingData 分页基本数据通用格式
type PagingData struct {
	List   []any `json:"list"`
	Paging Page  `json:"paging"`
}

// Apply runs a count query against db and fills in page.Total/PageCount.
func Apply(db *gorm.DB, page *Page) {
	var count int64
	db.Count(&count)
	page.Total = int(count)
	page.PageCount = int((page.Total + page.PageSize - 1) / page.PageSize)
}

// Limit 为查询加上当前页的 LIMIT / OFFSET; Current 为 0 时也视为第一页
func Limit(db *gorm.DB, page *Page) *gorm.DB {
	return db.Limit(page.PageSize).Offset(max(page.Current-1, 0) * page.PageSize)
}
