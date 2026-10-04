package ad

import (
	"strings"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestValidate(t *testing.T) {
	ok := Ad{Slot: "play_top", Name: "首页横幅", Image: "/upload/pic/a.gif", Link: "https://example.com/x?a=1"}
	if err := normalize(&ok); err != nil {
		t.Fatalf("valid ad rejected: %v", err)
	}
	relative := Ad{Slot: "play_bottom", Name: "x", Image: "https://img.test/b.webp", Link: "/filmDetail?link=1"}
	if err := normalize(&relative); err != nil {
		t.Fatalf("relative link rejected: %v", err)
	}
	for name, a := range map[string]Ad{
		"empty slot":    {Name: "x", Image: "/a.png"},
		"bad slot":      {Slot: "Play Top!", Name: "x", Image: "/a.png"},
		"no name":       {Slot: "s", Image: "/a.png"},
		"no image":      {Slot: "s", Name: "x"},
		"js link":       {Slot: "s", Name: "x", Image: "/a.png", Link: "javascript:alert(1)"},
		"data image":    {Slot: "s", Name: "x", Image: "data:image/png;base64,AAAA"},
		"js image":      {Slot: "s", Name: "x", Image: "javascript:alert(1)"},
		"negative sort": {Slot: "s", Name: "x", Image: "/a.png", Sort: -1},
		"protocol-rel":  {Slot: "s", Name: "x", Image: "/a.png", Link: "//evil.test"},
	} {
		if err := normalize(&a); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
	trim := Ad{Slot: " play_top ", Name: " 名称 ", Image: " /a.png ", Link: " "}
	if err := normalize(&trim); err != nil || trim.Slot != "play_top" || trim.Name != "名称" || trim.Image != "/a.png" || trim.Link != "" {
		t.Fatalf("trim = %+v %v", trim, err)
	}
}

// 前台只取该方案、该广告位中启用的广告, 按排序值
func TestSlotQuery(t *testing.T) {
	d, err := gorm.Open(mysql.New(mysql.Config{DSN: "u:p@tcp(127.0.0.1:1)/x", SkipInitializeWithVersion: true}), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}
	var l []Ad
	sql := slotQuery(d, 2, "play_top").Find(&l).Statement.SQL.String()
	for _, want := range []string{"scheme_id = ?", "slot = ?", "status = ?", "ORDER BY sort ASC, id ASC"} {
		if !strings.Contains(sql, want) {
			t.Fatalf("SQL missing %q: %s", want, sql)
		}
	}
}
