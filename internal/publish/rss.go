package publish

import (
	"bytes"
	"gomaccms/internal/config"
	"gomaccms/internal/film"
	"log"
	"regexp"
	"strings"
	"time"
)

// RSSChannel RSS 频道信息 (取自请求域名的站点设置)
type RSSChannel struct {
	Title       string
	Description string
}

// RSSService /rss.xml: 分类方案中最近更新的影片, 渲染结果按 分类方案 + 域名 缓存 (缓存不可用时直接查询)
type RSSService struct {
	cache  Cache
	latest func(schemeId int64, limit int) ([]film.Vod, error)
	limit  int
	ttl    time.Duration
	// langOf 方案的默认语言 (RSS 按它显示影片文字); nil 时不覆盖
	langOf func(schemeId int64) string
}

// newRSSService 正式环境: 最近更新的前台可见影片
func newRSSService(cache Cache) *RSSService {
	return &RSSService{
		cache: cache,
		latest: func(schemeId int64, limit int) ([]film.Vod, error) {
			return film.LatestVods(detailTypeIds(schemeId), limit)
		},
		limit:  config.RSSLimit,
		ttl:    config.RSSCacheTTL,
		langOf: func(id int64) string { return film.CategorySvc.Scheme(id).DefaultLang },
	}
}

// Feed 分类方案的 RSS (base 为 协议://域名)
func (s *RSSService) Feed(schemeId int64, base string, ch RSSChannel) ([]byte, error) {
	key := config.RSSCacheKeyOf(schemeId, base)
	if v, err := s.cache.Get(key); err == nil && v != "" {
		return []byte(v), nil
	}
	vods, err := s.latest(schemeId, s.limit)
	if err != nil {
		return nil, err
	}
	if s.langOf != nil {
		film.LocalizeVods(vods, s.langOf(schemeId))
	}
	b := renderRSS(base, ch, vods)
	if err := s.cache.Set(key, string(b), s.ttl); err != nil {
		log.Println("Cache RSS failed:", err)
	}
	return b, nil
}

// Invalidate 清除分类方案的 RSS 缓存 (各域名)
func (s *RSSService) Invalidate(schemeId int64) {
	if err := s.cache.DelPrefix(config.RSSSchemeKeyOf(schemeId)); err != nil {
		log.Println("Invalidate RSS cache failed:", err)
	}
}

var htmlTag = regexp.MustCompile(`<[^>]*>`)

// rssDescription 备注 + 去除 HTML 的简介 (最多 200 字)
func rssDescription(v film.Vod) string {
	text := strings.Join(strings.Fields(htmlTag.ReplaceAllString(v.VodContent, "")), " ")
	text = clip(text, 200)
	if v.VodRemarks != "" && text != "" {
		return v.VodRemarks + " - " + text
	}
	return v.VodRemarks + text
}

// renderRSS RSS 2.0
func renderRSS(base string, ch RSSChannel, vods []film.Vod) []byte {
	var b bytes.Buffer
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n" + `<rss version="2.0"><channel>`)
	b.WriteString("<title>" + xmlEscape(ch.Title) + "</title>")
	b.WriteString("<link>" + xmlEscape(base+HomePath) + "</link>")
	b.WriteString("<description>" + xmlEscape(ch.Description) + "</description>")
	b.WriteString("<lastBuildDate>" + time.Now().Format(time.RFC1123Z) + "</lastBuildDate>\n")
	for _, v := range vods {
		link := xmlEscape(base + VodPath(v.VodId))
		b.WriteString("<item><title>" + xmlEscape(v.VodName) + "</title><link>" + link + `</link><guid isPermaLink="true">` + link + "</guid>")
		if v.VodTime > 0 {
			b.WriteString("<pubDate>" + time.Unix(v.VodTime, 0).UTC().Format(time.RFC1123Z) + "</pubDate>")
		}
		b.WriteString("<description>" + xmlEscape(rssDescription(v)) + "</description></item>\n")
	}
	b.WriteString("</channel></rss>\n")
	return b.Bytes()
}
