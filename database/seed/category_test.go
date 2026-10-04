package seed

import "testing"

// 预设分类: ID 与 slug 不重复, 子分类的 ID 不与一级分类重复, 都小于后台新增分类的起始ID (10000)
func TestDefaultCategories(t *testing.T) {
	ids, slugs := map[int64]bool{}, map[string]bool{}
	check := func(id int64, slug string) {
		if id <= 0 || id >= 10000 || ids[id] || slug == "" || slugs[slug] {
			t.Fatalf("bad or duplicate category %d / %q", id, slug)
		}
		ids[id], slugs[slug] = true, true
	}
	for _, top := range defaultCategories {
		check(top.Id, top.Slug)
		for _, c := range top.Children {
			check(c.Id, c.Slug)
		}
	}
	if len(ids) != 15 {
		t.Fatalf("want 15 categories, got %d", len(ids))
	}
}
