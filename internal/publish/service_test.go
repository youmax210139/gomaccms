package publish

import (
	"net/http"
	"slices"
	"testing"
	"time"

	"gomaccms/internal/config"
)

// testService 不连接 MySQL / Redis 的发布中心: 分类方案 1、2, 每个方案一个域名; indexNowEndpoint 为空时 IndexNow 停用
func testService(cache *memCache, indexNowEndpoint string) (*Service, *int) {
	resets := 0
	n := NewIndexNow(IndexNowConfig{Enabled: indexNowEndpoint != "", Key: "k", Endpoint: indexNowEndpoint, BatchSize: 100}, http.DefaultClient)
	n.sleep = func(time.Duration) {}
	s := &Service{
		runner:       NewRunner(newMemStore()),
		cache:        cache,
		RSS:          &RSSService{cache: cache, latest: (&fakeLatest{}).fn, limit: 5, ttl: time.Minute},
		IndexNow:     n,
		resetMemory:  func() { resets++ },
		resetSearch:  func() error { return nil },
		schemeIds:    func() []int64 { return []int64{1, 2} },
		hostsOf:      func(id int64) []string { return []string{"s" + string(rune('0'+id)) + ".com"} },
		buildSitemap: func(id int64, p *Progress) error { return nil },
	}
	return s, &resets
}

func TestVodsChangedInvalidatesItsSchemes(t *testing.T) {
	cache := newMemCache()
	for _, id := range []int64{1, 2} {
		cache.Set(config.HomeCacheKeyOf(id), "home", 0)
		cache.Set(config.RSSCacheKeyOf(id, "https://a.com"), "rss", 0)
	}
	s, resets := testService(cache, "http://127.0.0.1:1")
	s.VodsChanged([]int64{3, 4}, []int64{1})
	if _, err := cache.Get(config.HomeCacheKeyOf(1)); err == nil {
		t.Fatal("scheme 1 home cache must be invalidated")
	}
	if _, err := cache.Get(config.RSSCacheKeyOf(1, "https://a.com")); err == nil {
		t.Fatal("scheme 1 RSS cache must be invalidated")
	}
	if _, err := cache.Get(config.HomeCacheKeyOf(2)); err != nil {
		t.Fatal("scheme 2 is not affected and keeps its cache")
	}
	if !s.SitemapDirty(1) || s.SitemapDirty(2) {
		t.Fatal("only scheme 1 sitemap must be marked dirty")
	}
	if *resets != 1 {
		t.Fatal("in-memory caches (today count, hot search) must be reset")
	}
	if s.IndexNow.Pending(1) != 2 || s.IndexNow.Pending(2) != 0 {
		t.Fatalf("changed films must be queued for their scheme: %d / %d", s.IndexNow.Pending(1), s.IndexNow.Pending(2))
	}
	// 影响范围不确定 (如分类停用): 全部方案清缓存、标记 sitemap, 不推送
	s.VodsChanged(nil, nil)
	if !s.SitemapDirty(2) || s.IndexNow.Pending(2) != 0 {
		t.Fatal("nil schemes means all schemes, without IndexNow")
	}
}

func TestVodsChangedRedisUnavailable(t *testing.T) {
	cache := newMemCache()
	cache.fail = true
	s, _ := testService(cache, "http://127.0.0.1:1")
	s.VodsChanged([]int64{1}, []int64{1}) // 不能 panic, 内容操作照常完成
	if s.IndexNow.Pending(1) != 1 {
		t.Fatal("IndexNow queue must not depend on Redis")
	}
}

func TestRefreshCache(t *testing.T) {
	cache := newMemCache()
	s, resets := testService(cache, "")
	for _, target := range []string{"home", "today", "rss", "search", "all"} {
		if err := s.RefreshCache(target); err != nil {
			t.Fatalf("%s: %v", target, err)
		}
	}
	if !slices.Contains(cache.deleted, config.IndexCacheKey) || !slices.Contains(cache.deleted, config.RSSCacheKey) {
		t.Fatalf("deleted = %v", cache.deleted)
	}
	if *resets != 2 { // today + all
		t.Fatalf("memory reset %d times, want 2", *resets)
	}
	if err := s.RefreshCache("nope"); err == nil {
		t.Fatal("unknown target must be rejected")
	}
}

func TestRefreshSchemeRSS(t *testing.T) {
	cache := newMemCache()
	cache.Set(config.RSSCacheKeyOf(1, "https://a.com"), "x", 0)
	cache.Set(config.RSSCacheKeyOf(2, "https://a.com"), "y", 0)
	s, _ := testService(cache, "")
	s.RefreshRSS(1)
	if _, err := cache.Get(config.RSSCacheKeyOf(1, "https://a.com")); err == nil {
		t.Fatal("scheme 1 RSS must be cleared")
	}
	if _, err := cache.Get(config.RSSCacheKeyOf(2, "https://a.com")); err != nil {
		t.Fatal("scheme 2 RSS must be kept")
	}
}

// TestSitemapJobPerScheme 每个分类方案各自重建; 同一方案重建中时返回进行中的任务
func TestSitemapJobPerScheme(t *testing.T) {
	s, _ := testService(newMemCache(), "")
	block := make(chan struct{})
	s.buildSitemap = func(id int64, p *Progress) error { <-block; return nil }
	s.cache.Set(config.SitemapDirtyKeyOf(1), "1", 0)
	a, created, err := s.RebuildSitemap(1)
	if err != nil || !created || a.SchemeId != 1 {
		t.Fatalf("first rebuild: %+v created=%v err=%v", a, created, err)
	}
	if s.SitemapDirty(1) {
		t.Fatal("starting a rebuild clears the scheme's dirty flag")
	}
	b, created, _ := s.RebuildSitemap(1)
	if created || b.Id != a.Id {
		t.Fatal("rebuilding the same scheme must return the running job")
	}
	c, created, _ := s.RebuildSitemap(2)
	if !created || c.Id == a.Id {
		t.Fatal("another scheme must rebuild independently")
	}
	close(block)
	s.runner.Wait()
}

func TestSubmitIndexNowPerScheme(t *testing.T) {
	api := &fakeIndexNowAPI{}
	s, _ := testService(newMemCache(), api.server(t).URL)
	s.VodsChanged([]int64{1, 2}, []int64{1})
	s.VodsChanged([]int64{9}, []int64{2})
	if _, _, err := s.SubmitIndexNow(1); err != nil {
		t.Fatal(err)
	}
	s.runner.Wait()
	if len(api.payloads) != 1 || api.payloads[0].Host != "s1.com" || len(api.payloads[0].URLList) != 2 {
		t.Fatalf("payloads = %+v", api.payloads)
	}
	if s.IndexNow.Pending(2) != 1 {
		t.Fatal("scheme 2 queue must be untouched")
	}
}
