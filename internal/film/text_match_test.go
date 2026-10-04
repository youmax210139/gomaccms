package film

import (
	"strings"
	"testing"

	"gomaccms/internal/db"
)

// useMariaDB 测试期间切换数据库类型
func useMariaDB(t *testing.T, on bool) {
	t.Helper()
	old := db.IsMariaDB
	db.IsMariaDB = on
	t.Cleanup(func() { db.IsMariaDB = old })
}

func matchSQL(t *testing.T, word string, phrase bool, columns ...string) (string, []any) {
	t.Helper()
	var ids []int64
	stmt := dryDB(t).Model(&Vod{}).Select("vod_id").Where(textMatch(word, phrase, columns...)).Find(&ids).Statement
	return stmt.SQL.String(), stmt.Vars
}

func TestTextMatchMySQL(t *testing.T) {
	useMariaDB(t, false)
	sql, vars := matchSQL(t, "流浪地球", true, "vod_name", "vod_sub")
	if !strings.Contains(sql, "MATCH(vod_name, vod_sub) AGAINST(? IN BOOLEAN MODE)") || vars[0] != `"流浪地球"` {
		t.Errorf("phrase: %s %v", sql, vars)
	}
	sql, vars = matchSQL(t, "动作 科幻", false, "vod_class")
	if !strings.Contains(sql, "MATCH(vod_class) AGAINST(?)") || strings.Contains(sql, "BOOLEAN") || vars[0] != "动作 科幻" {
		t.Errorf("natural: %s %v", sql, vars)
	}
}

func TestTextMatchMariaDB(t *testing.T) {
	useMariaDB(t, true)
	sql, vars := matchSQL(t, "流浪地球", true, "vod_name", "vod_sub")
	if strings.Contains(sql, "MATCH") || !strings.Contains(sql, "(vod_name LIKE ? OR vod_sub LIKE ?)") {
		t.Errorf("phrase SQL: %s", sql)
	}
	if len(vars) != 2 || vars[0] != "%流浪地球%" || vars[1] != "%流浪地球%" {
		t.Errorf("phrase vars: %v", vars)
	}
	// 自然语言模式: 空格分隔的任一词匹配
	sql, vars = matchSQL(t, "动作 科幻", false, "vod_class")
	if !strings.Contains(sql, "(vod_class LIKE ? OR vod_class LIKE ?)") || vars[0] != "%动作%" || vars[1] != "%科幻%" {
		t.Errorf("natural: %s %v", sql, vars)
	}
	// LIKE 通配符要转义
	_, vars = matchSQL(t, `100%_a\b`, true, "vod_name")
	if vars[0] != `%100\%\_a\\b%` {
		t.Errorf("escape: %v", vars[0])
	}
}

func TestKeywordQueryMariaDB(t *testing.T) {
	useMariaDB(t, true)
	var ids []int64
	sql := keywordQuery(dryDB(t), "Trái Đất", "vi").Find(&ids).Statement.SQL.String()
	if strings.Contains(sql, "MATCH") || !strings.Contains(sql, "name LIKE ?") || !strings.Contains(sql, "vod_i18n") {
		t.Errorf("translated keyword search on MariaDB:\n%s", sql)
	}
}

func TestRelateQueryMariaDB(t *testing.T) {
	useMariaDB(t, true)
	sql, vars := relateSQL(t)
	if strings.Contains(sql, "MATCH") || !strings.Contains(sql, "(vod_name LIKE ? OR vod_sub LIKE ?)") {
		t.Errorf("related films on MariaDB:\n%s", sql)
	}
	if vars[len(vars)-1] == "%2%" {
		t.Errorf("film name must not be split into short terms: %v", vars)
	}
}

func TestRelateQueryMySQL(t *testing.T) {
	useMariaDB(t, false)
	sql, _ := relateSQL(t)
	if !strings.Contains(sql, "ORDER BY MATCH(vod_name, vod_sub) AGAINST(?) DESC, vod_time DESC") {
		t.Errorf("MySQL keeps relevance order:\n%s", sql)
	}
}

// relateSQL 相关影片查询的 SQL ("流浪地球 2", 剧情 "科幻")
func relateSQL(t *testing.T) (string, []any) {
	t.Helper()
	old := db.Mdb
	db.Mdb = dryDB(t)
	t.Cleanup(func() { db.Mdb = old })
	var ids []int64
	stmt := relateQuery(SearchInfo{Name: "流浪地球 2", ClassTag: "科幻", Cid: 6}).Find(&ids).Statement
	return stmt.SQL.String(), stmt.Vars
}
