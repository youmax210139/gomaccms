package film

import (
	"strings"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// dryDB 只生成 SQL、不连接数据库的 gorm 实例
func dryDB(t *testing.T) *gorm.DB {
	t.Helper()
	d, err := gorm.Open(mysql.New(mysql.Config{DSN: "u:p@tcp(127.0.0.1:1)/x", SkipInitializeWithVersion: true}), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// TestSitemapVodQueryOnlyPublished sitemap / RSS 只包含已审核且在可见分类中的影片
func TestSitemapVodQueryOnlyPublished(t *testing.T) {
	var out []SitemapVod
	stmt := sitemapVodQuery(dryDB(t), []int64{10008, 10009}).Where("vod_id > ?", 5).Order("vod_id").Limit(100).Find(&out).Statement
	sql := stmt.SQL.String()
	for _, want := range []string{"vod_status = ?", "EXISTS (SELECT 1 FROM vod_type t WHERE t.vod_id = vod.vod_id AND t.type_id IN (?,?) AND t.status = ?)", "ORDER BY vod_id", "LIMIT ?"} {
		if !strings.Contains(sql, want) {
			t.Fatalf("SQL missing %q:\n%s", want, sql)
		}
	}
	if stmt.Vars[0] != vodStatusOn {
		t.Fatalf("first condition must require vod_status = %d, got %v", vodStatusOn, stmt.Vars[0])
	}
}
