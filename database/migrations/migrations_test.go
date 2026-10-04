package migrations

import "testing"

// fresh 删除表时的表名引用: 反引号括起, 名称中的反引号加倍
func TestQuoteIdent(t *testing.T) {
	for in, want := range map[string]string{"vod": "`vod`", "a`b": "`a``b`", "goose_db_version": "`goose_db_version`"} {
		if got := quoteIdent(in); got != want {
			t.Errorf("quoteIdent(%q) = %s, want %s", in, got, want)
		}
	}
}

// 内嵌的迁移文件都能被 goose 解析 (版本连续、没有重复)
func TestEmbeddedMigrations(t *testing.T) {
	sources, err := sources()
	if err != nil {
		t.Fatal(err)
	}
	for i, s := range sources {
		if s.Version != int64(i+1) {
			t.Fatalf("migration #%d has version %d (%s): versions must be 1..n without gaps", i+1, s.Version, s.Path)
		}
	}
	if len(sources) == 0 {
		t.Fatal("no migrations embedded")
	}
}
