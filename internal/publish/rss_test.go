package publish

import (
	"encoding/xml"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"gomaccms/internal/film"
)

// memCache 测试用的缓存; fail 为 true 时模拟 Redis 不可用
type memCache struct {
	mu      sync.Mutex
	data    map[string]string
	fail    bool
	deleted []string
}

func newMemCache() *memCache { return &memCache{data: map[string]string{}} }

func (m *memCache) Get(key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail {
		return "", errors.New("redis: connection refused")
	}
	v, ok := m.data[key]
	if !ok {
		return "", errors.New("redis: nil")
	}
	return v, nil
}

func (m *memCache) Set(key, val string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail {
		return errors.New("redis: connection refused")
	}
	m.data[key] = val
	return nil
}

func (m *memCache) DelPrefix(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deleted = append(m.deleted, key)
	if m.fail {
		return errors.New("redis: connection refused")
	}
	for k := range m.data {
		if k == key || strings.HasPrefix(k, key+":") {
			delete(m.data, k)
		}
	}
	return nil
}

// fakeLatest 返回 n 部影片, 并记录被调用的次数与 limit
type fakeLatest struct {
	calls, limit int
	err          error
}

func (f *fakeLatest) fn(schemeId int64, limit int) ([]film.Vod, error) {
	f.calls++
	f.limit = limit
	if f.err != nil {
		return nil, f.err
	}
	return []film.Vod{
		{VodId: 7, VodName: "Tom & Jerry <3>", VodRemarks: "更新至10集", VodContent: "<p>一部 <b>动画</b></p>", VodTime: time.Date(2026, 10, 3, 8, 0, 0, 0, time.UTC).Unix()},
		{VodId: 8, VodName: "第二部"},
	}, nil
}

type rss struct {
	Channel struct {
		Title string `xml:"title"`
		Link  string `xml:"link"`
		Items []struct {
			Title   string `xml:"title"`
			Link    string `xml:"link"`
			GUID    string `xml:"guid"`
			PubDate string `xml:"pubDate"`
			Desc    string `xml:"description"`
		} `xml:"item"`
	} `xml:"channel"`
}

func TestRSSFeedRendersAndCaches(t *testing.T) {
	cache, latest := newMemCache(), &fakeLatest{}
	s := &RSSService{cache: cache, latest: latest.fn, limit: 50, ttl: time.Minute}
	ch := RSSChannel{Title: "GoMacCMS", Description: "在线观影"}
	b, err := s.Feed(1, "https://a.com", ch)
	if err != nil {
		t.Fatal(err)
	}
	var doc rss
	if err := xml.Unmarshal(b, &doc); err != nil {
		t.Fatalf("invalid RSS XML: %v\n%s", err, b)
	}
	it := doc.Channel.Items
	if doc.Channel.Link != "https://a.com/index" || len(it) != 2 {
		t.Fatalf("channel = %+v", doc.Channel)
	}
	if it[0].Title != "Tom & Jerry <3>" || it[0].Link != "https://a.com/filmDetail?link=7" || it[0].GUID != it[0].Link {
		t.Fatalf("item = %+v", it[0])
	}
	if it[0].PubDate != "Sat, 03 Oct 2026 08:00:00 +0000" {
		t.Fatalf("pubDate = %q", it[0].PubDate)
	}
	if strings.Contains(it[0].Desc, "<p>") || !strings.Contains(it[0].Desc, "更新至10集") || !strings.Contains(it[0].Desc, "一部 动画") {
		t.Fatalf("description = %q", it[0].Desc)
	}
	if latest.limit != 50 {
		t.Fatalf("limit passed = %d", latest.limit)
	}
	// 第二次命中缓存, 不再查数据库
	if _, err := s.Feed(1, "https://a.com", ch); err != nil || latest.calls != 1 {
		t.Fatalf("second request must be served from cache: calls=%d err=%v", latest.calls, err)
	}
	// 不同域名分开缓存 (RSS 内是绝对 URL)
	if b2, _ := s.Feed(1, "https://b.com", ch); latest.calls != 2 || !strings.Contains(string(b2), "https://b.com/") {
		t.Fatal("another host must get its own feed")
	}
}

func TestRSSFeedRedisUnavailable(t *testing.T) {
	cache, latest := newMemCache(), &fakeLatest{}
	cache.fail = true
	s := &RSSService{cache: cache, latest: latest.fn, limit: 50, ttl: time.Minute}
	for i := 0; i < 2; i++ {
		if _, err := s.Feed(1, "https://a.com", RSSChannel{Title: "x"}); err != nil {
			t.Fatalf("Redis down must fall back to the database: %v", err)
		}
	}
	if latest.calls != 2 {
		t.Fatalf("without cache every request renders from DB: calls=%d", latest.calls)
	}
}

func TestRSSFeedDBError(t *testing.T) {
	s := &RSSService{cache: newMemCache(), latest: (&fakeLatest{err: errors.New("db down")}).fn, limit: 50, ttl: time.Minute}
	if _, err := s.Feed(1, "https://a.com", RSSChannel{}); err == nil {
		t.Fatal("database error must be reported")
	}
}

func TestRSSInvalidate(t *testing.T) {
	cache, latest := newMemCache(), &fakeLatest{}
	s := &RSSService{cache: cache, latest: latest.fn, limit: 50, ttl: time.Minute}
	s.Feed(1, "https://a.com", RSSChannel{})
	s.Invalidate(1)
	s.Feed(1, "https://a.com", RSSChannel{})
	if latest.calls != 2 {
		t.Fatalf("feed must be rebuilt after invalidation: calls=%d", latest.calls)
	}
}
