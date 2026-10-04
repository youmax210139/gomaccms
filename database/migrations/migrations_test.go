package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

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

// MariaDB 没有 ngram: 去掉 WITH PARSER ngram, 保留普通全文索引; MySQL 原样
func TestSourceFS(t *testing.T) {
	read := func(mariaDB bool) string {
		fsys, err := sourceFS(mariaDB)
		if err != nil {
			t.Fatal(err)
		}
		b, err := fs.ReadFile(fsys, "00001_schema.sql")
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	mysql, maria := read(false), read(true)
	if strings.Count(mysql, "WITH PARSER `ngram`") != 3 {
		t.Error("MySQL schema should keep the 3 ngram parsers")
	}
	if strings.Contains(maria, "ngram") {
		t.Error("MariaDB schema must not mention ngram")
	}
	if strings.Count(maria, "FULLTEXT KEY") != 3 {
		t.Error("MariaDB schema should keep the fulltext keys")
	}
	if names, _ := fs.Glob(must(sourceFS(true)), "*.sql"); len(names) == 0 {
		t.Error("MariaDB source FS should list the migrations")
	}
}

func must(f fs.FS, err error) fs.FS {
	if err != nil {
		panic(err)
	}
	return f
}
