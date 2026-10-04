package siteconfig

import (
	"strings"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestBannerTextFallback(t *testing.T) {
	b := Banner{Name: "庆余年", I18n: map[string]BannerText{"vi": {Name: "Khánh Dư Niên"}, "en": {Name: ""}}}
	if got := b.Text("vi"); got.Name != "Khánh Dư Niên" {
		t.Fatalf("vi text = %+v", got)
	}
	if got := b.Text("en"); got.Name != "庆余年" {
		t.Fatalf("empty translation must fall back to the main name, got %+v", got)
	}
	if got := b.Text("zh-CN"); got.Name != "庆余年" {
		t.Fatalf("source language = %+v", got)
	}
}

// TestBannerQuery 前台只取该分类方案已启用的海报, 按排序值
func TestBannerQuery(t *testing.T) {
	d, err := gorm.Open(mysql.New(mysql.Config{DSN: "u:p@tcp(127.0.0.1:1)/x", SkipInitializeWithVersion: true}), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}
	var l []Banner
	sql := bannerQuery(d, 2, true).Find(&l).Statement.SQL.String()
	for _, want := range []string{"scheme_id = ?", "status = ?", "ORDER BY sort ASC, id ASC"} {
		if !strings.Contains(sql, want) {
			t.Fatalf("SQL missing %q: %s", want, sql)
		}
	}
	if sql := bannerQuery(d, 2, false).Find(&l).Statement.SQL.String(); strings.Contains(sql, "status") {
		t.Fatalf("admin list must include disabled banners: %s", sql)
	}
}
