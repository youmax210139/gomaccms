package film

import (
	"strings"
	"testing"
)

// 后台挑选影片 (海报绑定): 只在该分类方案中, 关键字比对片名, 纯数字也比对影片ID
func TestSchemeVodQuery(t *testing.T) {
	var out []Vod
	sql := schemeVodQuery(dryDB(t), 2, "流浪").Find(&out).Statement.SQL.String()
	for _, want := range []string{"vod_type", "scheme_id = ?", "vod_name LIKE ?"} {
		if !strings.Contains(sql, want) {
			t.Fatalf("SQL missing %q:\n%s", want, sql)
		}
	}
	if strings.Contains(sql, "vod_id = ?") {
		t.Fatalf("a text keyword must not compare the ID:\n%s", sql)
	}
	stmt := schemeVodQuery(dryDB(t), 2, "555").Find(&out).Statement
	if !strings.Contains(stmt.SQL.String(), "vod_id = ? OR vod_name LIKE ?") {
		t.Fatalf("numeric keyword must also match the ID:\n%s", stmt.SQL.String())
	}
	if sql := schemeVodQuery(dryDB(t), 2, "").Find(&out).Statement.SQL.String(); strings.Contains(sql, "LIKE") {
		t.Fatalf("no keyword must not filter by name:\n%s", sql)
	}
}

func TestEscapeLike(t *testing.T) {
	if got := escapeLike(`50%_off\`); got != `50\%\_off\\` {
		t.Fatalf("escapeLike = %q", got)
	}
}
