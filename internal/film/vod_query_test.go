package film

import (
	"strings"
	"testing"
)

func TestVodQueryScope(t *testing.T) {
	var out []Vod
	q := VodQuery{SchemeId: 1, TypeIds: []int64{6}, Class: "动作", Area: "大陆", Lang: "国语", Year: "2019", Letter: "L",
		State: "正片", Ids: []int64{1, 2}, OrderBy: "hits", Desc: true, Offset: 20, Limit: 10}
	stmt := vodQueryScope(dryDB(t), q).Find(&out).Statement
	sql := stmt.SQL.String()
	for _, want := range []string{"vod_status = ?", "scheme_id = ?", "type_id IN (?) OR type_id_1 IN (?)", "vod_class LIKE ?",
		"vod_area = ?", "vod_lang = ?", "vod_year = ?", "vod_letter = ?", "vod_state = ?", "vod_id IN (?,?)",
		"ORDER BY vod_hits DESC", "LIMIT ?", "OFFSET ?"} {
		if !strings.Contains(sql, want) {
			t.Fatalf("SQL missing %q:\n%s", want, sql)
		}
	}
}

// 排序栏位只接受白名单, 其他值 (含 SQL 片段) 退回按更新时间
func TestVodQueryOrderWhitelist(t *testing.T) {
	var out []Vod
	sql := vodQueryScope(dryDB(t), VodQuery{SchemeId: 1, OrderBy: "vod_id; DROP TABLE vod", Limit: 10}).Find(&out).Statement.SQL.String()
	if strings.Contains(sql, "DROP") || !strings.Contains(sql, "ORDER BY vod_time ASC") {
		t.Fatalf("order must fall back to vod_time:\n%s", sql)
	}
	stmt := vodQueryScope(dryDB(t), VodQuery{SchemeId: 1, Year: "2010-2020", Class: "50%' OR 1=1 --", Limit: 1}).Find(&out).Statement
	if !strings.Contains(stmt.SQL.String(), "vod_year BETWEEN ? AND ?") {
		t.Fatalf("year range:\n%s", stmt.SQL.String())
	}
	found := false
	for _, v := range stmt.Vars {
		found = found || v == `%50\%' OR 1=1 --%`
	}
	if !found {
		t.Fatalf("LIKE value must be a bound, escaped parameter: %v", stmt.Vars)
	}
}
