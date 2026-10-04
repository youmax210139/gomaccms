// Package publish 网站地图: 每个分类方案的 sitemap (索引 + 分片)、RSS、IndexNow 推送, 以及缓存刷新.
// 页面一律由 Gin 即时渲染, 不生成静态 HTML; sitemap 由后台任务生成到 config.SitemapDir, 对外提供时补上请求的域名
package publish

import (
	"errors"
	"fmt"
	"gomaccms/internal/config"
	"gomaccms/internal/film"
	"gomaccms/internal/index"
	"gomaccms/internal/paging"
	"gomaccms/internal/theme"
	"log"
	"net/http"
	"slices"
	"time"
)

var Svc *Service

// Service 网站地图与缓存刷新
type Service struct {
	runner   *Runner
	cache    Cache
	RSS      *RSSService
	IndexNow *IndexNow
	// resetMemory 清除内存中的今日更新数与热搜词缓存; resetSearch 让关键字搜索结果缓存过期
	resetMemory func()
	resetSearch func() error
	// schemeIds 全部分类方案; hostsOf 分类方案的 IndexNow 推送域名; buildSitemap 重建分类方案的 sitemap
	schemeIds    func() []int64
	hostsOf      func(schemeId int64) []string
	buildSitemap func(schemeId int64, p *Progress) error
}

// NewService 使用 MySQL 任务表与 Redis 缓存
func NewService() *Service {
	cache := redisCache{}
	indexNow := NewIndexNow(IndexNowConfig{
		Enabled: config.IndexNowEnabled, Key: config.IndexNowKey, Scheme: config.IndexNowScheme,
		Endpoint: config.IndexNowEndpoint, BatchSize: config.IndexNowBatchSize,
	}, &http.Client{Timeout: 30 * time.Second})
	return &Service{
		runner: NewRunner(dbJobStore{}), cache: cache, RSS: newRSSService(cache), IndexNow: indexNow,
		resetMemory: index.Svc.ResetCaches, resetSearch: film.ResetSearchCache,
		schemeIds: func() []int64 {
			var ids []int64
			for _, sc := range film.CategorySvc.ListSchemes() {
				ids = append(ids, sc.Id)
			}
			return ids
		},
		hostsOf:      indexNowHosts,
		buildSitemap: BuildSitemap,
	}
}

// indexNowHosts 分类方案的推送域名: 使用该方案且网站开启的域名; 默认方案另加 INDEXNOW_HOST (未配置的域名)
func indexNowHosts(schemeId int64) []string {
	var hosts []string
	for _, d := range theme.Svc.AllDomains() {
		if d.State && d.SchemeOrDefault() == schemeId {
			hosts = append(hosts, d.Domain)
		}
	}
	if schemeId == film.DefaultSchemeId && config.IndexNowHost != "" && !slices.Contains(hosts, config.IndexNowHost) {
		hosts = append(hosts, config.IndexNowHost)
	}
	return hosts
}

// RebuildSitemap 开始重建分类方案 sitemap 的后台任务并立即返回; 该方案已在重建时返回进行中的任务
func (s *Service) RebuildSitemap(schemeId int64) (Job, bool, error) {
	job, created, err := s.runner.Start(JobSitemap, schemeId, func(p *Progress) error { return s.buildSitemap(schemeId, p) })
	if created {
		// 任务开始时清除「需要重建」标记, 之后的内容变动会再次标记
		if err := s.cache.Set(config.SitemapDirtyKeyOf(schemeId), "", time.Second); err != nil {
			log.Println("Clear sitemap dirty flag failed:", err)
		}
	}
	return job, created, err
}

// SitemapDirty 内容变动后分类方案的 sitemap 是否还没有重建
func (s *Service) SitemapDirty(schemeId int64) bool {
	v, err := s.cache.Get(config.SitemapDirtyKeyOf(schemeId))
	return err == nil && v != ""
}

// RefreshRSS 清除分类方案的 RSS 缓存, 下次请求重新生成
func (s *Service) RefreshRSS(schemeId int64) {
	s.RSS.Invalidate(schemeId)
}

// SubmitIndexNow 立即把分类方案待推送的 URL 提交到该方案的域名 (后台任务); 已在提交时返回进行中的任务
func (s *Service) SubmitIndexNow(schemeId int64) (Job, bool, error) {
	return s.runner.Start(JobIndexNow, schemeId, func(p *Progress) error {
		hosts := s.hostsOf(schemeId)
		p.SetTotal(int64(s.IndexNow.Pending(schemeId) * len(hosts)))
		_, err := s.IndexNow.Flush(schemeId, hosts, func(n int) { p.Add(int64(n)) })
		return err
	})
}

// Job 任务状态
func (s *Service) Job(id string) (*Job, error) {
	return s.runner.Get(id)
}

// Jobs 任务列表的一页 (新的在前), 同时设置 page 的总数与页数
func (s *Service) Jobs(page *paging.Page) []Job {
	l, total := s.runner.List((page.Current-1)*page.PageSize, page.PageSize)
	page.Total = int(total)
	page.PageCount = (page.Total + page.PageSize - 1) / page.PageSize
	return l
}

// VodsChanged 影片新增 / 修改 / 删除后调用 (film.VodsChanged): 清除受影响分类方案的首页与 RSS 缓存、
// 标记其 sitemap 需要重建、加入其 IndexNow 待推送; schemes 为 nil 时为全部方案
func (s *Service) VodsChanged(ids []int64, schemes []int64) {
	if schemes == nil {
		schemes = s.schemeIds()
	}
	paths := make([]string, len(ids))
	for i, id := range ids {
		paths[i] = VodPath(id)
	}
	for _, sc := range schemes {
		if err := s.cache.DelPrefix(config.HomeCacheKeyOf(sc)); err != nil {
			log.Println("Invalidate home cache failed:", err)
		}
		s.RSS.Invalidate(sc)
		if err := s.cache.Set(config.SitemapDirtyKeyOf(sc), "1", 0); err != nil {
			log.Println("Mark sitemap dirty failed:", err)
		}
		s.IndexNow.Enqueue(sc, paths...)
	}
	s.resetMemory()
}

// RefreshCache 后台手动刷新缓存: home 首页, today 今日更新数与热搜词, rss 全部 RSS, search 关键字搜索结果, all 全部.
// 页面本身即时渲染, 没有影片页 / 分类页缓存, 内容变动时也会自动失效 (见 VodsChanged)
func (s *Service) RefreshCache(target string) error {
	switch target {
	case "home":
		return s.cache.DelPrefix(config.IndexCacheKey)
	case "today":
		s.resetMemory()
		return nil
	case "rss":
		return s.cache.DelPrefix(config.RSSCacheKey)
	case "search":
		return s.resetSearch()
	case "all":
		s.resetMemory()
		return errors.Join(s.cache.DelPrefix(config.IndexCacheKey), s.cache.DelPrefix(config.RSSCacheKey), s.resetSearch())
	}
	return fmt.Errorf("未知的缓存: %s", target)
}

// Start 后台定时: 定时提交各分类方案的 IndexNow; 内容有变动的方案按 config.SitemapAutoInterval 自动重建 sitemap
func (s *Service) Start() {
	if s.IndexNow.Enabled() && config.IndexNowFlushInterval > 0 {
		go func() {
			for range time.Tick(config.IndexNowFlushInterval) {
				for _, sc := range s.schemeIds() {
					if s.IndexNow.Pending(sc) > 0 && !s.runner.Running(JobIndexNow, sc) {
						if _, err := s.IndexNow.Flush(sc, s.hostsOf(sc), nil); err != nil {
							log.Printf("IndexNow flush (scheme %d) failed: %v", sc, err)
						}
					}
				}
			}
		}()
	}
	if config.SitemapAutoInterval <= 0 {
		return
	}
	go func() {
		for range time.Tick(config.SitemapAutoInterval) {
			for _, sc := range s.schemeIds() {
				if s.SitemapDirty(sc) && !s.runner.Running(JobSitemap, sc) {
					if _, _, err := s.RebuildSitemap(sc); err != nil {
						log.Printf("Auto rebuild sitemap (scheme %d) failed: %v", sc, err)
					}
				}
			}
		}
	}()
}

// SchemeStatus 一个分类方案的网站地图状态
type SchemeStatus struct {
	Id       int64          `json:"id"`
	Name     string         `json:"name"`
	Hosts    []string       `json:"hosts"` // IndexNow 推送域名
	Sitemap  SitemapMeta    `json:"sitemap"`
	Dirty    bool           `json:"dirty"`
	IndexNow IndexNowStatus `json:"indexnow"`
}

// Status 网站地图页面显示的状态
type Status struct {
	Schemes         []SchemeStatus `json:"schemes"`
	ChunkSize       int            `json:"chunkSize"`
	AutoInterval    int            `json:"autoInterval"` // 分钟, 0 为不自动重建
	RSSLimit        int            `json:"rssLimit"`
	RSSTTL          int            `json:"rssTtl"` // 秒
	IndexNowEnabled bool           `json:"indexnowEnabled"`
	Jobs            []Job          `json:"jobs"`
	JobsPage        paging.Page    `json:"jobsPage"`
}

// Status 各分类方案的网站地图状态与任务列表的一页
func (s *Service) Status(page paging.Page) Status {
	st := Status{
		ChunkSize: config.SitemapChunkSize, AutoInterval: int(config.SitemapAutoInterval / time.Minute),
		RSSLimit: s.RSS.limit, RSSTTL: int(s.RSS.ttl / time.Second), IndexNowEnabled: s.IndexNow.Enabled(),
	}
	st.Jobs = s.Jobs(&page)
	st.JobsPage = page
	for _, sc := range film.CategorySvc.ListSchemes() {
		st.Schemes = append(st.Schemes, SchemeStatus{
			Id: sc.Id, Name: sc.Name, Hosts: s.hostsOf(sc.Id), Sitemap: readSitemapMeta(SchemeDir(sc.Id)),
			Dirty: s.SitemapDirty(sc.Id), IndexNow: s.IndexNow.Status(sc.Id),
		})
	}
	return st
}
