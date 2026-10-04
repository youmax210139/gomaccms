package film

import (
	"strings"
	"testing"
)

func TestKeywordQuerySourceLang(t *testing.T) {
	var ids []int64
	sql := keywordQuery(dryDB(t), "地球", "zh-CN").Find(&ids).Statement.SQL.String()
	if !strings.Contains(sql, "MATCH(vod_name, vod_sub)") || strings.Contains(sql, "vod_i18n") {
		t.Fatalf("source language must only search vod:\n%s", sql)
	}
	if vodSearchFieldOf("zh-CN") != "vod_name|vod_sub" || vodSearchFieldOf("") != "vod_name|vod_sub" {
		t.Fatal("source language keeps the old vod_search field")
	}
}

func TestKeywordQueryTranslatedLang(t *testing.T) {
	var ids []int64
	stmt := keywordQuery(dryDB(t), "Trái Đất", "vi").Find(&ids).Statement
	sql := stmt.SQL.String()
	for _, want := range []string{"vod_status = ?", "MATCH(vod_name, vod_sub)", "vod_i18n", "lang = ?", "MATCH(name, sub)"} {
		if !strings.Contains(sql, want) {
			t.Fatalf("SQL missing %q:\n%s", want, sql)
		}
	}
	if vodSearchFieldOf("vi") != "vod_name|vod_sub|vi" {
		t.Fatalf("field = %q", vodSearchFieldOf("vi"))
	}
}
