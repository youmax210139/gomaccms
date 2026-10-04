package publish

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// fakeIndexNowAPI 记录收到的请求; failTimes 次之前返回 500
type fakeIndexNowAPI struct {
	mu        sync.Mutex
	payloads  []indexNowPayload
	calls     int
	failTimes int
}

func (f *fakeIndexNowAPI) server(t *testing.T) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.calls++
		if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json; charset=utf-8" {
			t.Errorf("bad request: %s %s", r.Method, r.Header.Get("Content-Type"))
		}
		if f.calls <= f.failTimes {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var p indexNowPayload
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			t.Errorf("bad JSON: %v", err)
		}
		f.payloads = append(f.payloads, p)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func newTestIndexNow(endpoint string, enabled bool, batch int) *IndexNow {
	n := NewIndexNow(IndexNowConfig{Enabled: enabled, Key: "abc123", Scheme: "https", Endpoint: endpoint, BatchSize: batch}, http.DefaultClient)
	n.sleep = func(time.Duration) {}
	return n
}

func TestIndexNowDisabled(t *testing.T) {
	api := &fakeIndexNowAPI{}
	srv := api.server(t)
	n := newTestIndexNow(srv.URL, false, 100)
	n.Enqueue(1, "/filmDetail?link=1")
	if err := n.SubmitURLs("www.example.com", []string{"https://www.example.com/a"}); err != nil {
		t.Fatalf("disabled IndexNow must not error: %v", err)
	}
	if _, err := n.Flush(1, []string{"a.com"}, nil); err != nil {
		t.Fatal(err)
	}
	if api.calls != 0 || n.Pending(1) != 0 || n.Enabled() {
		t.Fatalf("disabled IndexNow must not queue or send: calls=%d pending=%d", api.calls, n.Pending(1))
	}
	// 启用但缺少 key 也视为停用
	m := NewIndexNow(IndexNowConfig{Enabled: true, Endpoint: srv.URL, BatchSize: 10}, http.DefaultClient)
	if m.Enabled() {
		t.Fatal("IndexNow without a key must be disabled")
	}
}

func TestIndexNowSubmitPayload(t *testing.T) {
	api := &fakeIndexNowAPI{}
	n := newTestIndexNow(api.server(t).URL, true, 100)
	if err := n.SubmitURL("www.example.com", "https://www.example.com/filmDetail?link=1"); err != nil {
		t.Fatal(err)
	}
	p := api.payloads[0]
	if p.Host != "www.example.com" || p.Key != "abc123" || p.KeyLocation != "https://www.example.com/abc123.txt" || len(p.URLList) != 1 {
		t.Fatalf("payload = %+v", p)
	}
}

// TestIndexNowSchemeQueues 每个分类方案一个待推送队列, 提交时向该方案的每个域名推送
func TestIndexNowSchemeQueues(t *testing.T) {
	api := &fakeIndexNowAPI{}
	n := newTestIndexNow(api.server(t).URL, true, 10)
	for i := 0; i < 25; i++ {
		n.Enqueue(1, VodPath(int64(i)))
	}
	n.Enqueue(1, VodPath(3)) // 重复的 URL 只提交一次
	n.Enqueue(2, VodPath(99))
	if n.Pending(1) != 25 || n.Pending(2) != 1 {
		t.Fatalf("pending = %d / %d, want 25 / 1", n.Pending(1), n.Pending(2))
	}
	processed := 0
	sent, err := n.Flush(1, []string{"a.com", "b.com"}, func(k int) { processed += k })
	if err != nil || sent != 50 || processed != 50 {
		t.Fatalf("flush: sent=%d processed=%d err=%v", sent, processed, err)
	}
	// 每个域名分 3 批 (10, 10, 5)
	if len(api.payloads) != 6 || len(api.payloads[0].URLList) != 10 || len(api.payloads[2].URLList) != 5 {
		t.Fatalf("batches = %d", len(api.payloads))
	}
	if api.payloads[0].Host != "a.com" || api.payloads[0].URLList[1] != "https://a.com/filmDetail?link=1" || api.payloads[3].Host != "b.com" {
		t.Fatalf("payload = %+v", api.payloads[0])
	}
	if n.Pending(1) != 0 || n.Pending(2) != 1 || n.Status(1).LastCount != 50 {
		t.Fatalf("after flush: pending=%d/%d status=%+v", n.Pending(1), n.Pending(2), n.Status(1))
	}
}

// TestIndexNowNoHosts 方案没有可推送的域名时报错, 队列保留 (设置域名后再提交)
func TestIndexNowNoHosts(t *testing.T) {
	api := &fakeIndexNowAPI{}
	n := newTestIndexNow(api.server(t).URL, true, 10)
	n.Enqueue(1, VodPath(1))
	if _, err := n.Flush(1, nil, nil); err == nil {
		t.Fatal("a scheme without hosts must report an error")
	}
	if n.Pending(1) != 1 || api.calls != 0 {
		t.Fatal("queue must be kept when there is no host")
	}
}

func TestIndexNowRetryThenSucceed(t *testing.T) {
	api := &fakeIndexNowAPI{failTimes: 2}
	n := newTestIndexNow(api.server(t).URL, true, 10)
	if err := n.SubmitURL("a.com", "https://a.com/a"); err != nil || api.calls != 3 {
		t.Fatalf("must retry and succeed on the 3rd attempt: calls=%d err=%v", api.calls, err)
	}
}

func TestIndexNowAPIFailure(t *testing.T) {
	api := &fakeIndexNowAPI{failTimes: 100}
	n := newTestIndexNow(api.server(t).URL, true, 10)
	n.Enqueue(1, "/filmDetail?link=1")
	if _, err := n.Flush(1, []string{"a.com"}, nil); err == nil {
		t.Fatal("API failure must be reported")
	}
	if api.calls != 3 {
		t.Fatalf("at most 3 attempts per batch, got %d", api.calls)
	}
	if st := n.Status(1); st.LastError == "" || n.Pending(1) != 0 {
		t.Fatalf("failure must be recorded and the batch dropped: %+v pending=%d", st, n.Pending(1))
	}
}
