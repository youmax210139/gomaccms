package publish

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"
)

// memStore 测试用的内存任务存储
type memStore struct {
	mu   sync.Mutex
	jobs map[string]Job
}

func newMemStore() *memStore { return &memStore{jobs: map[string]Job{}} }

func (m *memStore) Create(j *Job) error { return m.Save(j) }
func (m *memStore) Save(j *Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jobs[j.Id] = *j
	return nil
}
func (m *memStore) Get(id string) (*Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, ok := m.jobs[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return &j, nil
}
func (m *memStore) List(offset, limit int) ([]Job, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var l []Job
	for _, j := range m.jobs {
		l = append(l, j)
	}
	sort.Slice(l, func(a, b int) bool { return l[a].CreatedAt.After(l[b].CreatedAt) })
	total := int64(len(l))
	l = l[min(offset, len(l)):]
	return l[:min(limit, len(l))], total, nil
}
func (m *memStore) FailActive(msg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, j := range m.jobs {
		if j.Status == JobPending || j.Status == JobRunning {
			j.Status, j.Error = JobFailed, msg
			m.jobs[id] = j
		}
	}
	return nil
}

func TestRunnerSingleJobPerType(t *testing.T) {
	store := newMemStore()
	r := NewRunner(store)
	release := make(chan struct{})
	started := make(chan struct{})
	a, created, err := r.Start("sitemap", 1, func(p *Progress) error {
		p.SetTotal(10)
		p.Add(4)
		close(started)
		<-release
		p.Add(6)
		return nil
	})
	if err != nil || !created {
		t.Fatalf("first start: created=%v err=%v", created, err)
	}
	<-started
	b, created, err := r.Start("sitemap", 1, func(p *Progress) error { t.Fatal("second job must not run"); return nil })
	if err != nil || created || b.Id != a.Id {
		t.Fatalf("second start must return the running job: %+v created=%v err=%v", b, created, err)
	}
	if b.Status != JobRunning || b.Total != 10 || b.Processed != 4 {
		t.Fatalf("running snapshot = %+v", b)
	}
	// 其他类型、其他分类方案不受影响
	c, created, _ := r.Start("indexnow", 1, func(p *Progress) error { return nil })
	if !created || c.Id == a.Id {
		t.Fatal("a different job type must start its own job")
	}
	e, created, _ := r.Start("sitemap", 2, func(p *Progress) error { return nil })
	if !created || e.Id == a.Id || e.SchemeId != 2 {
		t.Fatal("another scheme must start its own sitemap job")
	}
	close(release)
	r.Wait()
	done, _ := r.Get(a.Id)
	if done.Status != JobCompleted || done.Processed != 10 || done.StartedAt == nil || done.FinishedAt == nil {
		t.Fatalf("finished job = %+v", done)
	}
	// 完成后可以再开始新的
	d, created, _ := r.Start("sitemap", 1, func(p *Progress) error { return nil })
	r.Wait()
	if !created || d.Id == a.Id {
		t.Fatal("a new job must start after the previous one finished")
	}
}

func TestRunnerFailedJob(t *testing.T) {
	r := NewRunner(newMemStore())
	j, _, _ := r.Start("sitemap", 1, func(p *Progress) error { return errors.New("permission denied") })
	r.Wait()
	got, _ := r.Get(j.Id)
	if got.Status != JobFailed || got.Error != "permission denied" {
		t.Fatalf("failed job = %+v", got)
	}
}

func TestRunnerPanicFailsJob(t *testing.T) {
	r := NewRunner(newMemStore())
	j, _, _ := r.Start("sitemap", 1, func(p *Progress) error { panic("boom") })
	r.Wait()
	got, _ := r.Get(j.Id)
	if got.Status != JobFailed {
		t.Fatalf("panicking job must be failed: %+v", got)
	}
	if _, created, _ := r.Start("sitemap", 1, func(p *Progress) error { return nil }); !created {
		t.Fatal("runner must accept new jobs after a panic")
	}
	r.Wait()
}

func TestRunnerRecoverInterrupted(t *testing.T) {
	store := newMemStore()
	now := time.Now()
	store.Save(&Job{Id: "x", Type: "sitemap", Status: JobRunning, CreatedAt: now})
	NewRunner(store) // 启动时把上次未结束的任务标记为失败
	j, _ := store.Get("x")
	if j.Status != JobFailed || j.Error == "" {
		t.Fatalf("interrupted job = %+v", j)
	}
}

// 任务列表分页: 最新的在前, 返回总数
func TestRunnerListPages(t *testing.T) {
	store := newMemStore()
	base := time.Now()
	for i := 0; i < 25; i++ {
		_ = store.Create(&Job{Id: fmt.Sprintf("j%02d", i), Type: JobSitemap, Status: JobCompleted, CreatedAt: base.Add(time.Duration(i) * time.Second)})
	}
	r := NewRunner(store)
	l, total := r.List(20, 10)
	if total != 25 || len(l) != 5 {
		t.Fatalf("page 3 = %d jobs, total %d; want 5 / 25", len(l), total)
	}
	if l[0].Id != "j04" || l[4].Id != "j00" {
		t.Fatalf("page 3 = %s..%s, want j04..j00 (newest first)", l[0].Id, l[4].Id)
	}
	if first, _ := r.List(0, 10); first[0].Id != "j24" {
		t.Fatalf("page 1 starts with %s, want the newest j24", first[0].Id)
	}
}
