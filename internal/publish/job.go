package publish

import (
	"fmt"
	"gomaccms/internal/db"
	"gomaccms/internal/util"
	"log"
	"sync"
	"time"
)

// 任务状态
const (
	JobPending   = "pending"
	JobRunning   = "running"
	JobCompleted = "completed"
	JobFailed    = "failed"
)

// 任务类型
const (
	JobSitemap  = "sitemap"
	JobIndexNow = "indexnow"
)

// Job 发布中心的后台任务 (publish_jobs 表)
type Job struct {
	Id         string     `json:"id" gorm:"primaryKey"`
	Type       string     `json:"type"`
	SchemeId   int64      `json:"schemeId"` // 分类方案
	Status     string     `json:"status"`
	Total      int64      `json:"total"`
	Processed  int64      `json:"processed"`
	Error      string     `json:"error"`
	StartedAt  *time.Time `json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

// TableName 任务表表名
func (Job) TableName() string {
	return "publish_jobs"
}

// JobStore 任务的存储 (正式环境为 MySQL, 测试为内存)
type JobStore interface {
	Create(j *Job) error
	Save(j *Job) error
	Get(id string) (*Job, error)
	// List 任务列表 (新的在前) 的一页与任务总数
	List(offset, limit int) ([]Job, int64, error)
	// FailActive 把未结束的任务标记为失败 (服务重启时)
	FailActive(msg string) error
}

// dbJobStore MySQL 中的任务
type dbJobStore struct{}

func (dbJobStore) Create(j *Job) error { return db.Mdb.Create(j).Error }
func (dbJobStore) Save(j *Job) error   { return db.Mdb.Save(j).Error }
func (dbJobStore) Get(id string) (*Job, error) {
	var j Job
	if err := db.Mdb.First(&j, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &j, nil
}
func (dbJobStore) List(offset, limit int) ([]Job, int64, error) {
	var total int64
	if err := db.Mdb.Model(&Job{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var l []Job
	err := db.Mdb.Order("created_at DESC").Offset(offset).Limit(limit).Find(&l).Error
	return l, total, err
}
func (dbJobStore) FailActive(msg string) error {
	return db.Mdb.Model(&Job{}).Where("status IN ?", []string{JobPending, JobRunning}).
		Updates(map[string]any{"status": JobFailed, "error": msg, "finished_at": time.Now()}).Error
}

// Progress 运行中任务的进度 (由任务函数更新, 每秒最多写一次存储)
type Progress struct {
	r     *Runner
	job   *Job
	saved time.Time
}

// SetTotal 设置总数
func (p *Progress) SetTotal(n int64) {
	p.r.mu.Lock()
	p.job.Total = n
	p.r.mu.Unlock()
	p.save(false)
}

// Add 已处理数增加 n
func (p *Progress) Add(n int64) {
	p.r.mu.Lock()
	p.job.Processed += n
	p.r.mu.Unlock()
	p.save(false)
}

func (p *Progress) save(force bool) {
	if !force && time.Since(p.saved) < time.Second {
		return
	}
	p.saved = time.Now()
	p.r.mu.Lock()
	j := *p.job
	p.r.mu.Unlock()
	if err := p.r.store.Save(&j); err != nil {
		log.Println("Save publish job failed:", err)
	}
}

// Runner 在后台执行任务; 同一类型 + 分类方案同时只有一个任务, 避免两个任务同时覆写同一批文件
type Runner struct {
	store  JobStore
	mu     sync.Mutex
	active map[string]*Job // 类型:分类方案 → 进行中的任务
	wg     sync.WaitGroup
}

// activeKey 进行中任务的 key
func activeKey(jobType string, schemeId int64) string {
	return fmt.Sprintf("%s:%d", jobType, schemeId)
}

// NewRunner 建立任务执行器, 并把上次服务停止时未结束的任务标记为失败
func NewRunner(store JobStore) *Runner {
	if err := store.FailActive("服务重启, 任务中断"); err != nil {
		log.Println("Recover publish jobs failed:", err)
	}
	return &Runner{store: store, active: map[string]*Job{}}
}

// Start 在后台开始分类方案的一个任务并立即返回; 已有同类进行中的任务时返回它 (created 为 false), 不重复执行
func (r *Runner) Start(jobType string, schemeId int64, run func(p *Progress) error) (job Job, created bool, err error) {
	key := activeKey(jobType, schemeId)
	r.mu.Lock()
	if j, ok := r.active[key]; ok {
		snap := *j
		r.mu.Unlock()
		return snap, false, nil
	}
	j := &Job{Id: util.GenerateSalt(), Type: jobType, SchemeId: schemeId, Status: JobPending, CreatedAt: time.Now()}
	if err := r.store.Create(j); err != nil {
		r.mu.Unlock()
		return Job{}, false, err
	}
	r.active[key] = j
	snap := *j
	r.mu.Unlock()

	r.wg.Add(1)
	go r.run(j, run)
	return snap, true, nil
}

func (r *Runner) run(j *Job, run func(p *Progress) error) {
	defer r.wg.Done()
	p := &Progress{r: r, job: j}
	r.mu.Lock()
	now := time.Now()
	j.Status, j.StartedAt = JobRunning, &now
	r.mu.Unlock()
	p.save(true)

	err := func() (err error) {
		defer func() {
			if v := recover(); v != nil {
				err = fmt.Errorf("任务异常: %v", v)
			}
		}()
		return run(p)
	}()

	r.mu.Lock()
	end := time.Now()
	j.FinishedAt = &end
	if err != nil {
		j.Status, j.Error = JobFailed, clip(err.Error(), 1000)
		log.Printf("publish job %s (%s) failed: %v", j.Id, j.Type, err)
	} else {
		j.Status = JobCompleted
	}
	delete(r.active, activeKey(j.Type, j.SchemeId))
	r.mu.Unlock()
	p.save(true)
}

// Get 任务状态 (进行中的任务返回内存中的最新进度)
func (r *Runner) Get(id string) (*Job, error) {
	r.mu.Lock()
	for _, j := range r.active {
		if j.Id == id {
			snap := *j
			r.mu.Unlock()
			return &snap, nil
		}
	}
	r.mu.Unlock()
	return r.store.Get(id)
}

// List 任务列表的一页 (新的在前, 进行中的以内存中的进度为准) 与任务总数
func (r *Runner) List(offset, limit int) ([]Job, int64) {
	l, total, err := r.store.List(offset, limit)
	if err != nil {
		log.Println("List publish jobs failed:", err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range l {
		if j, ok := r.active[activeKey(l[i].Type, l[i].SchemeId)]; ok && j.Id == l[i].Id {
			l[i] = *j
		}
	}
	return l, total
}

// Running 分类方案是否有该类型进行中的任务
func (r *Runner) Running(jobType string, schemeId int64) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.active[activeKey(jobType, schemeId)]
	return ok
}

// Wait 等待所有进行中的任务结束
func (r *Runner) Wait() {
	r.wg.Wait()
}

// clip 截取前 n 个字符
func clip(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}
