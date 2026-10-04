package publish

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

// IndexNow 推送的最多尝试次数与每个分类方案待推送 URL 的上限 (超出时丢弃, sitemap 仍会收录)
const (
	indexNowAttempts   = 3
	indexNowMaxPending = 100000
)

// IndexNowConfig IndexNow 设置 (见 config.IndexNow*)
type IndexNowConfig struct {
	Enabled   bool
	Key       string
	Scheme    string // 推送 URL 的协议
	Endpoint  string
	BatchSize int
}

// IndexNowStatus 一个分类方案的推送状态
type IndexNowStatus struct {
	Pending    int       `json:"pending"`
	LastSubmit time.Time `json:"lastSubmit"`
	LastCount  int       `json:"lastCount"`
	LastError  string    `json:"lastError"`
}

type indexNowPayload struct {
	Host        string   `json:"host"`
	Key         string   `json:"key"`
	KeyLocation string   `json:"keyLocation"`
	URLList     []string `json:"urlList"`
}

// indexNowQueue 一个分类方案待推送的站内路径
type indexNowQueue struct {
	paths []string
	seen  map[string]bool
	IndexNowStatus
}

// IndexNow 把变动的 URL 通知搜索引擎: 影片变动时按分类方案加入待推送, 定时或手动提交到该方案的每个域名;
// 失败只记录, 不影响内容操作
type IndexNow struct {
	cfg    IndexNowConfig
	client *http.Client
	sleep  func(time.Duration)

	mu     sync.Mutex
	queues map[int64]*indexNowQueue
}

// NewIndexNow 未启用或缺少 key 时为停用状态 (只记录日志, 不报错)
func NewIndexNow(cfg IndexNowConfig, client *http.Client) *IndexNow {
	if cfg.Enabled && cfg.Key == "" {
		log.Println("IndexNow disabled: INDEXNOW_KEY not set")
		cfg.Enabled = false
	} else if !cfg.Enabled {
		log.Println("IndexNow disabled")
	}
	if cfg.BatchSize <= 0 || cfg.BatchSize > 10000 {
		cfg.BatchSize = 10000
	}
	if cfg.Scheme == "" {
		cfg.Scheme = "https"
	}
	return &IndexNow{cfg: cfg, client: client, sleep: time.Sleep, queues: map[int64]*indexNowQueue{}}
}

// Enabled 是否启用
func (n *IndexNow) Enabled() bool {
	return n.cfg.Enabled
}

// URL 域名 + 站内路径
func (n *IndexNow) URL(host, path string) string {
	return n.cfg.Scheme + "://" + host + path
}

// KeyFile 验证文件 /<key>.txt 的路径与内容 (停用时为空); 每个域名都提供
func (n *IndexNow) KeyFile() (path, content string) {
	if !n.cfg.Enabled {
		return "", ""
	}
	return "/" + n.cfg.Key + ".txt", n.cfg.Key
}

// queue 分类方案的队列 (调用方持有锁)
func (n *IndexNow) queue(schemeId int64) *indexNowQueue {
	q, ok := n.queues[schemeId]
	if !ok {
		q = &indexNowQueue{seen: map[string]bool{}}
		n.queues[schemeId] = q
	}
	return q
}

// Enqueue 加入分类方案待推送的站内路径 (去重)
func (n *IndexNow) Enqueue(schemeId int64, paths ...string) {
	if !n.cfg.Enabled || len(paths) == 0 {
		return
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	q := n.queue(schemeId)
	for _, p := range paths {
		if q.seen[p] || len(q.paths) >= indexNowMaxPending {
			continue
		}
		q.seen[p] = true
		q.paths = append(q.paths, p)
	}
}

// Pending 分类方案待推送的 URL 数
func (n *IndexNow) Pending(schemeId int64) int {
	n.mu.Lock()
	defer n.mu.Unlock()
	if q, ok := n.queues[schemeId]; ok {
		return len(q.paths)
	}
	return 0
}

// Flush 把分类方案待推送的路径提交到 hosts 的每个域名; progress 为每批提交后的回调.
// 没有域名时报错并保留队列; 失败的批次记录后丢弃 (sitemap 仍会收录)
func (n *IndexNow) Flush(schemeId int64, hosts []string, progress func(int)) (int, error) {
	if !n.cfg.Enabled {
		return 0, nil
	}
	n.mu.Lock()
	q := n.queue(schemeId)
	paths := q.paths
	if len(paths) == 0 {
		n.mu.Unlock()
		return 0, nil
	}
	if len(hosts) == 0 {
		n.mu.Unlock()
		return 0, errors.New("该分类方案没有可推送的域名")
	}
	q.paths, q.seen = nil, map[string]bool{}
	n.mu.Unlock()

	sent := 0
	var errs []error
	for _, host := range hosts {
		urls := make([]string, len(paths))
		for i, p := range paths {
			urls[i] = n.URL(host, p)
		}
		k, err := n.submit(host, urls, progress)
		sent += k
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", host, err))
		}
	}
	err := errors.Join(errs...)
	n.mu.Lock()
	q.LastSubmit, q.LastCount, q.LastError = time.Now(), sent, ""
	if err != nil {
		q.LastError = err.Error()
	}
	n.mu.Unlock()
	return sent, err
}

// SubmitURL 立即向 host 提交一个 URL
func (n *IndexNow) SubmitURL(host, url string) error {
	return n.SubmitURLs(host, []string{url})
}

// SubmitURLs 立即向 host 提交 URL (按 BatchSize 分批); 停用时不做任何事
func (n *IndexNow) SubmitURLs(host string, urls []string) error {
	if !n.cfg.Enabled {
		return nil
	}
	_, err := n.submit(host, urls, nil)
	return err
}

func (n *IndexNow) submit(host string, urls []string, progress func(int)) (int, error) {
	sent := 0
	var lastErr error
	for start := 0; start < len(urls); start += n.cfg.BatchSize {
		batch := urls[start:min(start+n.cfg.BatchSize, len(urls))]
		if err := n.post(host, batch); err != nil {
			lastErr = err
			log.Printf("IndexNow submit %d URLs to %s failed: %v", len(batch), host, err)
		} else {
			sent += len(batch)
		}
		if progress != nil {
			progress(len(batch))
		}
	}
	return sent, lastErr
}

// post 提交一批, 失败时最多重试到 indexNowAttempts 次
func (n *IndexNow) post(host string, batch []string) error {
	body, _ := json.Marshal(indexNowPayload{Host: host, Key: n.cfg.Key, KeyLocation: n.URL(host, "/"+n.cfg.Key+".txt"), URLList: batch})
	var err error
	for attempt := 1; attempt <= indexNowAttempts; attempt++ {
		if attempt > 1 {
			n.sleep(time.Duration(attempt-1) * 2 * time.Second)
		}
		var resp *http.Response
		resp, err = n.client.Post(n.cfg.Endpoint, "application/json; charset=utf-8", bytes.NewReader(body))
		if err != nil {
			continue
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		// 200 OK / 202 Accepted 为成功
		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusAccepted {
			return nil
		}
		err = fmt.Errorf("IndexNow HTTP %d", resp.StatusCode)
	}
	return err
}

// Status 分类方案的推送状态
func (n *IndexNow) Status(schemeId int64) IndexNowStatus {
	n.mu.Lock()
	defer n.mu.Unlock()
	st := n.queue(schemeId).IndexNowStatus
	st.Pending = len(n.queue(schemeId).paths)
	return st
}
