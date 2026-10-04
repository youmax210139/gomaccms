package publish

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// vodFeed 测试用的影片迭代器: 共 n 部, 每次返回 batch 部 (路径 /v/<i>)
func vodFeed(n, batch int, failAt int) func() ([]SitemapEntry, error) {
	next := 0
	return func() ([]SitemapEntry, error) {
		if failAt > 0 && next >= failAt {
			return nil, errors.New("db error")
		}
		var out []SitemapEntry
		for i := 0; i < batch && next < n; i++ {
			next++
			out = append(out, SitemapEntry{Path: fmt.Sprintf("/v/%d", next), LastMod: time.Date(2026, 10, 3, 8, 0, 0, 0, time.UTC)})
		}
		return out, nil
	}
}

type urlset struct {
	URLs []struct {
		Loc     string `xml:"loc"`
		LastMod string `xml:"lastmod"`
	} `xml:"url"`
}

type sitemapIndex struct {
	Maps []struct {
		Loc string `xml:"loc"`
	} `xml:"sitemap"`
}

// serve 以 base 读取已生成的文件并解析
func serve(t *testing.T, dir, name, base string, v any) {
	t.Helper()
	var buf bytes.Buffer
	if err := ServeSitemapFile(&buf, dir, name, base); err != nil {
		t.Fatalf("serve %s: %v", name, err)
	}
	if err := xml.Unmarshal(buf.Bytes(), v); err != nil {
		t.Fatalf("%s is not valid XML: %v\n%s", name, err, buf.String())
	}
}

func TestWriteSitemapsShardsAndIndex(t *testing.T) {
	dir := t.TempDir()
	pages := []SitemapEntry{{Path: "/index"}, {Path: "/filmClassify?Pid=1&x=<y>"}}
	meta, err := writeSitemaps(dir, 3, pages, vodFeed(7, 2, 0), nil)
	if err != nil {
		t.Fatal(err)
	}
	if meta.VodURLs != 7 || meta.Shards != 3 || meta.PageURLs != 2 {
		t.Fatalf("meta = %+v, want 7 vod URLs in 3 shards", meta)
	}
	var idx sitemapIndex
	serve(t, dir, "sitemap.xml", "https://a.com", &idx)
	var locs []string
	for _, m := range idx.Maps {
		locs = append(locs, m.Loc)
	}
	want := []string{"https://a.com/sitemap-page.xml", "https://a.com/sitemap-vod-1.xml", "https://a.com/sitemap-vod-2.xml", "https://a.com/sitemap-vod-3.xml"}
	if strings.Join(locs, ",") != strings.Join(want, ",") {
		t.Fatalf("index locs = %v, want %v", locs, want)
	}
	// 每个索引指向的分片都存在
	for _, m := range idx.Maps {
		name := strings.TrimPrefix(m.Loc, "https://a.com/")
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("index points to missing shard %s", name)
		}
	}
	var s3 urlset
	serve(t, dir, "sitemap-vod-3.xml", "https://b.org", &s3)
	if len(s3.URLs) != 1 || s3.URLs[0].Loc != "https://b.org/v/7" {
		t.Fatalf("last shard = %+v", s3.URLs)
	}
	if s3.URLs[0].LastMod != "2026-10-03T08:00:00Z" {
		t.Fatalf("lastmod = %q", s3.URLs[0].LastMod)
	}
	var pg urlset
	serve(t, dir, "sitemap-page.xml", "https://a.com", &pg)
	if pg.URLs[1].Loc != "https://a.com/filmClassify?Pid=1&x=<y>" || pg.URLs[0].LastMod != "" {
		t.Fatalf("page sitemap must escape the URL and omit an unknown lastmod: %+v", pg.URLs)
	}
}

// TestWriteSitemaps100k 10 万部影片按 50000 分成 2 个分片
func TestWriteSitemaps100k(t *testing.T) {
	dir := t.TempDir()
	meta, err := writeSitemaps(dir, 50000, nil, vodFeed(100000, 1000, 0), nil)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Shards != 2 || meta.VodURLs != 100000 {
		t.Fatalf("meta = %+v", meta)
	}
	var s2 urlset
	serve(t, dir, "sitemap-vod-2.xml", "https://a.com", &s2)
	if len(s2.URLs) != 50000 {
		t.Fatalf("shard 2 has %d URLs", len(s2.URLs))
	}
}

// TestWriteSitemapsFailureKeepsPublished 生成失败时已发布的 sitemap 不变, 也不留下临时文件
func TestWriteSitemapsFailureKeepsPublished(t *testing.T) {
	dir := t.TempDir()
	if _, err := writeSitemaps(dir, 2, nil, vodFeed(4, 2, 0), nil); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(filepath.Join(dir, "sitemap.xml"))
	if _, err := writeSitemaps(dir, 2, nil, vodFeed(10, 2, 6), nil); err == nil {
		t.Fatal("feed error must fail the build")
	}
	after, _ := os.ReadFile(filepath.Join(dir, "sitemap.xml"))
	if !bytes.Equal(before, after) {
		t.Fatal("failed build must not change the published index")
	}
	tmps, _ := filepath.Glob(filepath.Join(dir, "*.tmp"))
	if len(tmps) != 0 {
		t.Fatalf("temporary files left: %v", tmps)
	}
}

// TestWriteSitemapsRemovesStaleShards 影片变少后, 多余的旧分片在新索引发布后删除
func TestWriteSitemapsRemovesStaleShards(t *testing.T) {
	dir := t.TempDir()
	if _, err := writeSitemaps(dir, 2, nil, vodFeed(6, 2, 0), nil); err != nil {
		t.Fatal(err)
	}
	if _, err := writeSitemaps(dir, 2, nil, vodFeed(2, 2, 0), nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "sitemap-vod-2.xml")); !os.IsNotExist(err) {
		t.Fatal("stale shard sitemap-vod-2.xml must be removed")
	}
}

func TestWriteSitemapsProgress(t *testing.T) {
	total := 0
	if _, err := writeSitemaps(t.TempDir(), 10, nil, vodFeed(25, 10, 0), func(n int) { total += n }); err != nil {
		t.Fatal(err)
	}
	if total != 25 {
		t.Fatalf("progress reported %d, want 25", total)
	}
}

func TestServeSitemapFileRejectsBadNames(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"../x.xml", "sitemap-vod-a.xml", "meta.json", "sitemap.xml.tmp"} {
		if err := ServeSitemapFile(&bytes.Buffer{}, dir, name, "https://a.com"); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("%q must be rejected as not found, got %v", name, err)
		}
	}
}
