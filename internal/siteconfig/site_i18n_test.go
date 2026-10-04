package siteconfig

import "testing"

func TestSiteInfoLocalized(t *testing.T) {
	si := SiteInfo{SiteName: "金丝雀", SeoTitle: "金丝雀影院", Keyword: "电影", Describe: "在线观影", LegalInfo: "版权所有", Logo: "/l.png",
		I18n: map[string]SiteText{"vi": {SiteName: "Hoàng Yến", Describe: "Xem phim"}}}
	vi := si.Localized("vi")
	if vi.SiteName != "Hoàng Yến" || vi.Describe != "Xem phim" || vi.SeoTitle != "金丝雀影院" || vi.LegalInfo != "版权所有" || vi.Logo != "/l.png" {
		t.Fatalf("vi = %+v", vi)
	}
	if si.SiteName != "金丝雀" {
		t.Fatal("Localized must not modify the original")
	}
	if zh := si.Localized("zh-CN"); zh.SiteName != "金丝雀" {
		t.Fatalf("zh = %+v", zh)
	}
}

func TestHintOf(t *testing.T) {
	c := BasicConfig{Hint: "网站维护中", HintI18n: map[string]string{"vi": "Đang bảo trì"}}
	if c.HintOf("vi") != "Đang bảo trì" || c.HintOf("en") != "网站维护中" || c.HintOf("") != "网站维护中" {
		t.Fatalf("HintOf = %q / %q", c.HintOf("vi"), c.HintOf("en"))
	}
}

func TestNormalizeSiteI18n(t *testing.T) {
	m := NormalizeSiteI18n(map[string]SiteText{"vi": {SiteName: " Tên ", LegalInfo: "  "}, "en": {SeoTitle: " "}})
	if len(m) != 1 || m["vi"].SiteName != "Tên" || m["vi"].LegalInfo != "" {
		t.Fatalf("normalized = %+v", m)
	}
	h := NormalizeHintI18n(map[string]string{"vi": " Đóng ", "en": "  "})
	if len(h) != 1 || h["vi"] != "Đóng" {
		t.Fatalf("hints = %+v", h)
	}
}
