package spider

import (
	"errors"
	"gomaccms/internal/collect"
	"gomaccms/internal/config"
	"gomaccms/internal/cron"
	"gomaccms/internal/film"
	"log"
)

var Svc = &Service{}

type Service struct{}

// StartCollect 开始对指定采集接口的采集任务 (后台执行); 该接口正在采集时返回错误
func (s *Service) StartCollect(id string, h int) error {
	return StartCollect(id, h)
}

// StopCollect 中止采集接口正在进行的采集任务
func (s *Service) StopCollect(id string) error {
	return StopCollect(id)
}

// ResumeCollect 从采集接口上次 (已中止 / 已中断 / 失败) 采集的中断处继续
func (s *Service) ResumeCollect(id string) error {
	return ResumeCollect(id)
}

// CollectByIds 采集指定采集站中指定ID的影片 (ids 以逗号分隔)
func (s *Service) CollectByIds(id string, ids string) error {
	fs := collect.Svc.GetFilmSource(id)
	if fs == nil {
		return errors.New("采集任务开启失败采集站信息不存在")
	}
	go collectFilmById(ids, fs)
	return nil
}

// SyncCollect 按影片的来源重新采集指定的本站影片 (ids 为本站影片ID, 逗号分隔)
func (s *Service) SyncCollect(ids string) {
	go CollectSingleFilm(ids)
}

// RetryFailedPages 重采一笔采集记录中失败的页
func (s *Service) RetryFailedPages(logId uint64) error {
	return RetryFailedPages(logId)
}

// RunCronTask 按定时任务类型执行对应的采集操作, 由 cron.TaskRunner 在任务触发时调用
func RunCronTask(ft cron.FilmCollectTask) {
	switch ft.Model {
	case 0:
		AutoCollect(ft.Time)
		log.Println("执行一次已启用站点的视频自动更新任务")
	case 1:
		// 对指定ids的资源站数据进行更新操作
		BatchCollect(ft.Time, ft.Ids...)
		log.Println("执行一次指定站点的视频自动更新任务")
	case 2:
		// 旧版的「采集重试」: 失效记录已并入采集记录 (按次重试失败的页), 启动时会删除此类任务
		log.Println("采集重试任务已废弃, 请在采集记录中重试失败的页")
	case 3:
		// 旧版的「影片信息同步」: 采集数据已直接入库, 不再需要同步
		log.Println("视频信息同步任务已废弃, 可在定时任务中删除")
	}
}

// ClearCache 清理API接口数据缓存
func ClearCache() {
	film.SearchRepo.RemoveCache(config.IndexCacheKey)
}
