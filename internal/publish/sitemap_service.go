package publish

import (
	"fmt"
	"gomaccms/internal/config"
	"gomaccms/internal/film"
	"path/filepath"
	"time"
)

// sitemapBatch 每次从数据库读取的影片数 (按 vod_id 递增分批, 不一次载入全部影片)
const sitemapBatch = 1000

// SchemeDir 分类方案的 sitemap 目录
func SchemeDir(schemeId int64) string {
	return filepath.Join(config.SitemapDir, fmt.Sprint(schemeId))
}

// sitemapPages 分类方案中可收录的页面: 首页、今日更新、排行榜、游客可看的一级分类页
func sitemapPages(schemeId int64) []SitemapEntry {
	pages := []SitemapEntry{{Path: HomePath}, {Path: TodayPath}, {Path: RankPath}}
	for _, pid := range listPids(schemeId) {
		pages = append(pages, SitemapEntry{Path: ClassifyPath(pid)})
	}
	return pages
}

// vodFeed 分类方案中前台可见的影片, 按 vod_id 逐批返回
func vodFeedOf(typeIds []int64) func() ([]SitemapEntry, error) {
	var after int64
	return func() ([]SitemapEntry, error) {
		l, err := film.SitemapVodBatch(typeIds, after, sitemapBatch)
		if err != nil || len(l) == 0 {
			return nil, err
		}
		out := make([]SitemapEntry, len(l))
		for i, v := range l {
			out[i] = SitemapEntry{Path: VodPath(v.VodId)}
			if v.VodTime > 0 {
				out[i].LastMod = time.Unix(v.VodTime, 0)
			}
		}
		after = l[len(l)-1].VodId
		return out, nil
	}
}

// BuildSitemap 重建一个分类方案的 sitemap (sitemap 任务的内容)
func BuildSitemap(schemeId int64, p *Progress) error {
	typeIds := detailTypeIds(schemeId)
	total, err := film.CountSitemapVods(typeIds)
	if err != nil {
		return fmt.Errorf("统计影片失败: %w", err)
	}
	p.SetTotal(total)
	_, err = writeSitemaps(SchemeDir(schemeId), config.SitemapChunkSize, sitemapPages(schemeId), vodFeedOf(typeIds),
		func(n int) { p.Add(int64(n)) })
	if err != nil {
		return fmt.Errorf("sitemap 生成失败: %w", err)
	}
	return nil
}
