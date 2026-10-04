package seed

import (
	"encoding/json"
	"log"

	"gomaccms/internal/config"
	"gomaccms/internal/db"
	"gomaccms/internal/film"

	"gorm.io/gorm/clause"
)

// defaultCategories 默认方案的预设视频分类 (与 MacCMS 默认分类相同的 ID 与名称): 一级分类与其子分类
var defaultCategories = []struct {
	film.Category
	Children []film.Category
}{
	{film.Category{Id: 1, Name: "电影", Slug: "dianying", Sort: 1}, []film.Category{
		{Id: 6, Name: "动作片", Slug: "dongzuopian", Sort: 1}, {Id: 7, Name: "喜剧片", Slug: "xijupian", Sort: 2},
		{Id: 8, Name: "爱情片", Slug: "aiqingpian", Sort: 3}, {Id: 9, Name: "科幻片", Slug: "kehuanpian", Sort: 4},
		{Id: 10, Name: "恐怖片", Slug: "kongbupian", Sort: 5}, {Id: 11, Name: "剧情片", Slug: "juqingpian", Sort: 6},
		{Id: 12, Name: "战争片", Slug: "zhanzhengpian", Sort: 7},
	}},
	{film.Category{Id: 2, Name: "连续剧", Slug: "lianxuju", Sort: 2}, []film.Category{
		{Id: 13, Name: "国产剧", Slug: "guochanju", Sort: 1}, {Id: 14, Name: "港台剧", Slug: "gangtaiju", Sort: 2},
		{Id: 15, Name: "日韩剧", Slug: "rihanju", Sort: 3}, {Id: 16, Name: "欧美剧", Slug: "oumeiju", Sort: 4},
	}},
	{film.Category{Id: 3, Name: "综艺", Slug: "zongyi", Sort: 3}, nil},
	{film.Category{Id: 4, Name: "动漫", Slug: "dongman", Sort: 4}, nil},
}

// seedCategories 默认方案还没有视频分类时写入分类: 有旧版存于 Redis 的分类树 (config.CategoryTreeKey) 时导入它
// (保留原分类ID、名称、显示状态与排序, Redis 中的旧数据不删除), 否则写入预设分类 (defaultCategories)
func seedCategories() error {
	if film.CategoryRepo.ExistsCategoryTree() {
		return nil
	}
	data := db.Rdb.Get(db.Cxt, config.CategoryTreeKey).Val()
	if data == "" {
		return seedDefaultCategories()
	}
	var tree film.CategoryTree
	if err := json.Unmarshal([]byte(data), &tree); err != nil {
		log.Println("seed: skip legacy category tree:", err)
		return nil
	}
	var cl []film.Category
	var walk func(t *film.CategoryTree)
	walk = func(t *film.CategoryTree) {
		for _, c := range t.Children {
			cl = append(cl, film.Category{Id: c.Id, Pid: c.Pid, Type: film.CategoryVideo, Name: c.Name,
				Slug: film.DefaultSlug(c.Id), Show: c.Show, Sort: c.Sort})
			walk(c)
		}
	}
	walk(&tree)
	if len(cl) == 0 {
		return nil
	}
	log.Printf("seed: importing %d categories from the legacy Redis category tree", len(cl))
	return db.Mdb.Clauses(clause.OnConflict{DoNothing: true}).Create(&cl).Error
}

// seedDefaultCategories 写入预设视频分类 (默认方案, 全部显示)
func seedDefaultCategories() error {
	var cl []film.Category
	for _, top := range defaultCategories {
		c := top.Category
		c.SchemeId, c.Type, c.Show = film.DefaultSchemeId, film.CategoryVideo, true
		cl = append(cl, c)
		for _, sub := range top.Children {
			sub.SchemeId, sub.Pid, sub.Type, sub.Show = film.DefaultSchemeId, top.Id, film.CategoryVideo, true
			cl = append(cl, sub)
		}
	}
	log.Printf("seed: %d default categories", len(cl))
	return db.Mdb.Clauses(clause.OnConflict{DoNothing: true}).Create(&cl).Error
}
