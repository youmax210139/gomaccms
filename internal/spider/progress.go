package spider

import (
	"encoding/json"
	"errors"
	"fmt"
	"gomaccms/internal/collect"
	"gomaccms/internal/db"
	"log"
	"slices"
	"sync"
	"time"
)

// Progress 一个采集接口最近一次采集的进度 (参照苹果 CMS: 当前分类、第几页 / 共几页、共多少条, 以及逐部视频的入库结果).
// 内存中一份, 同时存入 Redis (progressKey): 服务重启后进行中的采集改为「已中断」, 可从中断处继续 (ResumeTypeId / ResumePage)
type Progress struct {
	SourceId     string               `json:"sourceId"`
	SourceName   string               `json:"sourceName"`
	Mode         string               `json:"mode"`    // 采集当天 / 采集本周 / 采集所有 / 最近 N 小时 / 按ID采集, 续采时加「继续」前缀
	Trigger      string               `json:"trigger"` // 手动 / 定时任务 / 续采 / 重试
	Hours        int                  `json:"hours"`   // 采集时长参数 h (负数为全部), 续采时沿用
	State        string               `json:"state"`   // running / done / failed / stopped / interrupted
	Message      string               `json:"message"`
	StartedAt    int64                `json:"startedAt"`
	FinishedAt   int64                `json:"finishedAt"`
	TypeIndex    int                  `json:"typeIndex"` // 当前是第几个采集站分类 (从 1 开始)
	TypeCount    int                  `json:"typeCount"`
	TypeId       int64                `json:"typeId"`
	TypeName     string               `json:"typeName"`
	Page         int                  `json:"page"`      // 当前分类已采集的页数
	PageCount    int                  `json:"pageCount"` // 当前分类的总页数
	Total        int                  `json:"total"`     // 当前分类的数据总数
	Added        int                  `json:"added"`
	Updated      int                  `json:"updated"`
	Skipped      int                  `json:"skipped"`
	FailedPages  int                  `json:"failedPages"`
	FailedList   []collect.FailedPage `json:"failedList"`   // 失败的页, 最多 maxFailedList 个, 采集记录中可重试
	Logs         []string             `json:"logs"`         // 最近的日志, 最多 maxProgressLogs 条
	Stopping     bool                 `json:"stopping"`     // 已请求中止, 正在进行的页采集完后停止
	ResumeTypeId int64                `json:"resumeTypeId"` // 续采起点: 分类 (0 表示无需 / 无法续采)
	ResumePage   int                  `json:"resumePage"`   // 续采起点: 页码 (之前的页都已采集完)
	donePages    map[int]bool
}

const (
	progressRunning     = "running"
	progressDone        = "done"
	progressFailed      = "failed"
	progressStopped     = "stopped"
	progressInterrupted = "interrupted"
	maxProgressLogs     = 300
	maxFailedList       = 1000
	progressKey         = "Collect:Progress"

	TriggerManual = "手动"
	TriggerCron   = "定时任务"
	TriggerResume = "续采"
	TriggerRetry  = "重试"
)

var (
	progressMu   sync.Mutex
	progresses   = map[string]*Progress{}
	progressLoad sync.Once
)

// loadProgresses 从 Redis 载入上次的采集进度; 上次服务停止时仍在进行的采集记为「已中断」(调用方持有锁)
func loadProgresses() {
	progressLoad.Do(func() {
		for id, data := range db.Rdb.HGetAll(db.Cxt, progressKey).Val() {
			var p Progress
			if err := json.Unmarshal([]byte(data), &p); err != nil {
				continue
			}
			if p.State == progressRunning {
				p.State, p.Stopping, p.FinishedAt = progressInterrupted, false, time.Now().Unix()
				p.Message = fmt.Sprintf("服务重启, 采集已中断 (新增 %d, 更新 %d, 跳过 %d), 可从中断处继续", p.Added, p.Updated, p.Skipped)
				p.logf("%s", p.Message)
				saveProgress(&p)
				saveCollectLog(&p)
			}
			progresses[id] = &p
		}
	})
}

// saveProgress 将进度存入 Redis
func saveProgress(p *Progress) {
	data, _ := json.Marshal(p)
	if err := db.Rdb.HSet(db.Cxt, progressKey, p.SourceId, data).Err(); err != nil {
		log.Println("Save collect progress failed:", err)
	}
}

// saveCollectLog 采集结束时记录采集历史
func saveCollectLog(p *Progress) {
	collect.Repo.SaveCollectLog(collect.CollectLog{
		SourceId: p.SourceId, SourceName: p.SourceName, Mode: p.Mode, Trigger: p.Trigger, Hours: p.Hours, State: p.State,
		Added: p.Added, Updated: p.Updated, Skipped: p.Skipped, FailedPages: p.FailedPages, FailedList: p.FailedList, Message: p.Message,
		StartedAt: time.Unix(p.StartedAt, 0), FinishedAt: time.Unix(p.FinishedAt, 0),
	})
}

// modeLabel 采集时长 h 对应的采集方式名称
func modeLabel(h int) string {
	switch {
	case h < 0:
		return "采集所有"
	case h == 24:
		return "采集当天"
	case h == 24*7:
		return "采集本周"
	default:
		return fmt.Sprintf("最近 %d 小时", h)
	}
}

// beginProgress 开始记录一次采集; 同一采集接口正在采集时返回错误
func beginProgress(s *collect.FilmSource, mode, trigger string, h int) error {
	progressMu.Lock()
	defer progressMu.Unlock()
	loadProgresses()
	if p, ok := progresses[s.Id]; ok && p.State == progressRunning {
		return errors.New("该采集接口正在采集中, 请等待本次采集完成")
	}
	p := &Progress{SourceId: s.Id, SourceName: s.Name, Mode: mode, Trigger: trigger, Hours: h, State: progressRunning, StartedAt: time.Now().Unix()}
	progresses[s.Id] = p
	saveProgress(p)
	return nil
}

// trackProgress 更新正在进行的采集进度 (没有进行中的采集时忽略)
func trackProgress(id string, fn func(p *Progress)) {
	progressMu.Lock()
	defer progressMu.Unlock()
	if p, ok := progresses[id]; ok && p.State == progressRunning {
		fn(p)
		saveProgress(p)
	}
}

// startType 开始采集一个采集站分类, 从 startPage 页开始 (调用方持有锁)
func (p *Progress) startType(index int, typeId int64, name string, startPage, pageCount, total int) {
	p.TypeIndex, p.TypeId, p.TypeName = index, typeId, name
	p.Page, p.PageCount, p.Total = startPage-1, pageCount, total
	p.ResumeTypeId, p.ResumePage, p.donePages = typeId, startPage, map[int]bool{}
}

// pageDone 当前分类的一页采集完成 (含失败的页, 另记入 FailedList); 续采起点前进到第一个未完成的页 (调用方持有锁)
func (p *Progress) pageDone(pg int) {
	p.Page++
	if p.donePages == nil {
		p.donePages = map[int]bool{}
	}
	p.donePages[pg] = true
	for p.donePages[p.ResumePage] {
		delete(p.donePages, p.ResumePage)
		p.ResumePage++
	}
}

// pageFailed 记下采集失败的页 (调用方持有锁)
func (p *Progress) pageFailed(typeId int64, pg int) {
	p.FailedPages++
	if len(p.FailedList) < maxFailedList {
		p.FailedList = append(p.FailedList, collect.FailedPage{TypeId: typeId, Page: pg})
	}
}

// finishProgress 结束采集并记录采集历史; err 不为空时标记为失败
func finishProgress(id string, err error) {
	progressMu.Lock()
	defer progressMu.Unlock()
	p, ok := progresses[id]
	if !ok || p.State != progressRunning {
		return
	}
	p.FinishedAt = time.Now().Unix()
	switch {
	case err != nil:
		p.State, p.Message = progressFailed, err.Error()
	case p.Stopping:
		p.State = progressStopped
		p.Message = fmt.Sprintf("采集已中止: 新增 %d, 更新 %d, 跳过 %d", p.Added, p.Updated, p.Skipped)
	default:
		p.State = progressDone
		p.Message = fmt.Sprintf("采集完成: 新增 %d, 更新 %d, 跳过 %d", p.Added, p.Updated, p.Skipped)
		p.ResumeTypeId, p.ResumePage = 0, 0
	}
	if p.FailedPages > 0 {
		p.Message += fmt.Sprintf(", 失败 %d 页 (可在采集记录中重试)", p.FailedPages)
	}
	p.Stopping = false
	p.logf("%s", p.Message)
	saveProgress(p)
	saveCollectLog(p)
}

// logf 追加一条日志 (调用方已持有锁)
func (p *Progress) logf(format string, args ...any) {
	p.Logs = append(p.Logs, time.Now().Format("15:04:05 ")+fmt.Sprintf(format, args...))
	if n := len(p.Logs); n > maxProgressLogs {
		p.Logs = slices.Clone(p.Logs[n-maxProgressLogs:])
	}
}

// IsCollecting 采集接口是否正在采集
func IsCollecting(id string) bool {
	progressMu.Lock()
	defer progressMu.Unlock()
	loadProgresses()
	p, ok := progresses[id]
	return ok && p.State == progressRunning
}

// Progresses 全部采集接口最近一次的采集进度
func Progresses() []Progress {
	progressMu.Lock()
	defer progressMu.Unlock()
	loadProgresses()
	l := make([]Progress, 0, len(progresses))
	for _, p := range progresses {
		c := *p
		c.Logs = slices.Clone(p.Logs)
		c.donePages = nil
		l = append(l, c)
	}
	return l
}

// StopCollect 请求中止采集接口正在进行的采集任务 (正在进行的页采集完后停止)
func StopCollect(id string) error {
	progressMu.Lock()
	defer progressMu.Unlock()
	p, ok := progresses[id]
	if !ok || p.State != progressRunning {
		return errors.New("该采集接口没有正在进行的采集任务")
	}
	if !p.Stopping {
		p.Stopping = true
		p.logf("收到中止请求, 正在进行的页采集完后停止")
		saveProgress(p)
	}
	return nil
}

// stopRequested 采集任务是否已被请求中止
func stopRequested(id string) bool {
	progressMu.Lock()
	defer progressMu.Unlock()
	p, ok := progresses[id]
	return ok && p.State == progressRunning && p.Stopping
}

// resumePoint 续采的起点
type resumePoint struct {
	TypeId int64
	Page   int
}

// lastResumePoint 采集接口上次 (已中止 / 已中断 / 失败) 采集的续采起点与采集方式
func lastResumePoint(id string) (*resumePoint, *Progress, error) {
	progressMu.Lock()
	defer progressMu.Unlock()
	loadProgresses()
	p, ok := progresses[id]
	switch {
	case !ok:
		return nil, nil, errors.New("该采集接口没有可继续的采集")
	case p.State == progressRunning:
		return nil, nil, errors.New("该采集接口正在采集中")
	case p.State == progressDone || p.ResumeTypeId == 0 || p.Mode == "按ID采集":
		return nil, nil, errors.New("上次采集已完成或无法继续, 请重新采集")
	}
	c := *p
	return &resumePoint{TypeId: p.ResumeTypeId, Page: max(p.ResumePage, 1)}, &c, nil
}
