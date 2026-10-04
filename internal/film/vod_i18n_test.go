package film

import "testing"

func TestLocalizeBasicsPartial(t *testing.T) {
	list := []MovieBasicInfo{{Id: 1, Name: "流浪地球", Actor: "吴京", Blurb: "简介", Remarks: "HD"}, {Id: 2, Name: "长安"}}
	localizeBasics(list, map[int64]VodText{1: {Name: "Địa Cầu Lưu Lạc", Content: "Tóm tắt"}}, nil)
	if list[0].Name != "Địa Cầu Lưu Lạc" || list[0].Blurb != "Tóm tắt" || list[0].Actor != "吴京" || list[0].Remarks != "HD" {
		t.Fatalf("film 1 = %+v", list[0])
	}
	if list[1].Name != "长安" {
		t.Fatalf("untranslated film changed: %+v", list[1])
	}
}

func TestLocalizeBasicsCategoryName(t *testing.T) {
	list := []MovieBasicInfo{{Id: 1, Cid: 6, CName: "动作片"}, {Id: 2, Cid: 7, CName: "喜剧片"}}
	localizeBasics(list, nil, map[int64]string{6: "Phim hành động"})
	if list[0].CName != "Phim hành động" || list[1].CName != "喜剧片" {
		t.Fatalf("category names = %q, %q", list[0].CName, list[1].CName)
	}
}

func TestLocalizeSearchAndDetail(t *testing.T) {
	text := VodText{Name: "Tên", Class: "Hành động", LangText: "Tiếng Trung", Area: "Trung Quốc", Writer: "Tác giả"}
	s := []SearchInfo{{Mid: 3, Name: "名", ClassTag: "动作", Language: "国语", Area: "大陆"}}
	localizeSearchInfos(s, map[int64]VodText{3: text}, nil)
	if s[0].Name != "Tên" || s[0].ClassTag != "Hành động" || s[0].Language != "Tiếng Trung" || s[0].Area != "Trung Quốc" {
		t.Fatalf("search = %+v", s[0])
	}
	d := MovieDetail{Mid: 3, Name: "名", Writer: "作者", Content: "简介"}
	text.applyDetail(&d)
	if d.Name != "Tên" || d.Writer != "Tác giả" || d.Content != "简介" {
		t.Fatalf("detail = %+v", d)
	}
	v := Vod{VodName: "名", VodContent: "简介"}
	text.applyVod(&v)
	if v.VodName != "Tên" || v.VodContent != "简介" {
		t.Fatalf("vod = %+v", v)
	}
}

func TestLocalizeSourceLangNoQuery(t *testing.T) {
	// 原文语言不查询数据库 (db.Mdb 在测试中为 nil, 一旦查询就会 panic)
	if VodTexts([]int64{1}, "zh-CN") != nil || VodTexts([]int64{1}, "") != nil {
		t.Fatal("source language must not load translations")
	}
	LocalizeBasics([]MovieBasicInfo{{Id: 1}}, "zh-CN")
}

func TestSplitVodTranslations(t *testing.T) {
	save, del := splitVodTranslations(7, map[string]VodText{
		"vi":    {Name: " Tên ", Content: "  "},
		"en":    {Name: " ", Actor: ""},
		"zh-CN": {Name: "原文不能存"},
	}, nil)
	if len(save) != 1 || save[0].Lang != "vi" || save[0].Name != "Tên" || save[0].Content != "" || !save[0].Manual || save[0].VodId != 7 {
		t.Fatalf("save = %+v", save)
	}
	if len(del) != 1 || del[0] != "en" {
		t.Fatalf("del = %v", del)
	}
}

// 内容没有变的语言不重新保存 (保留原有的人工 / 自动标记)
func TestSplitVodTranslationsSkipsUnchanged(t *testing.T) {
	stored := map[string]VodI18n{"en": {VodId: 7, Lang: "en", VodText: VodText{Name: "Auto"}, Manual: false}}
	save, del := splitVodTranslations(7, map[string]VodText{"en": {Name: "Auto"}, "vi": {Name: "Tên"}}, stored)
	if len(save) != 1 || save[0].Lang != "vi" || len(del) != 0 {
		t.Fatalf("save = %+v, del = %v", save, del)
	}
}
