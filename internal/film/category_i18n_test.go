package film

import "testing"

func TestCategoryLocalize(t *testing.T) {
	c := &Category{Name: "电影", I18n: map[string]CategoryText{"vi": {Name: "Phim lẻ"}}}
	c.Localize("vi")
	if c.Name != "Phim lẻ" || c.SourceName != "电影" {
		t.Fatalf("localized = %+v", c)
	}
	d := &Category{Name: "综艺", I18n: map[string]CategoryText{"vi": {Name: ""}}}
	d.Localize("vi") // 译名留空时用原文
	if d.Name != "综艺" || d.SourceName != "综艺" {
		t.Fatalf("empty translation = %+v", d)
	}
	e := &Category{Name: "动漫"}
	e.Localize("en") // 没有译文
	if e.Name != "动漫" || e.SourceName != "动漫" {
		t.Fatalf("untranslated = %+v", e)
	}
	var nilCat *Category
	nilCat.Localize("vi") // 不能 panic
}

func TestCategoryTreeLocalize(t *testing.T) {
	tree := &CategoryTree{Category: &Category{Name: "根"}, Children: []*CategoryTree{
		{Category: &Category{Name: "电影", I18n: map[string]CategoryText{"vi": {Name: "Phim"}}}, Children: []*CategoryTree{
			{Category: &Category{Name: "喜剧", I18n: map[string]CategoryText{"vi": {Name: "Hài"}}}},
		}},
	}}
	tree.Localize("vi")
	if tree.Children[0].Name != "Phim" || tree.Children[0].Children[0].Name != "Hài" {
		t.Fatal("tree must be localized recursively")
	}
}

func TestNormalizeCategoryI18n(t *testing.T) {
	m := normalizeCategoryI18n(map[string]CategoryText{"vi": {Name: " Phim "}, "en": {Name: "  "}})
	if len(m) != 1 || m["vi"].Name != "Phim" {
		t.Fatalf("normalized = %+v", m)
	}
	if normalizeCategoryI18n(nil) != nil {
		t.Fatal("nil stays nil")
	}
}

// 保存时请求里没有的语言 (如方案暂时停用的语言) 保留原有译文; 请求里清空的语言删除
func TestMergeCategoryI18nKeepsMissingLanguages(t *testing.T) {
	old := map[string]CategoryText{"vi": {Name: "Phim"}, "en": {Name: "Movies"}}
	m := mergeCategoryI18n(old, map[string]CategoryText{"vi": {}})
	if _, ok := m["vi"]; ok {
		t.Fatalf("cleared vi must be removed: %+v", m)
	}
	if m["en"].Name != "Movies" {
		t.Fatalf("en (not in request) must be kept: %+v", m)
	}
	if old["vi"].Name != "Phim" {
		t.Fatal("old map must not be modified")
	}
}
