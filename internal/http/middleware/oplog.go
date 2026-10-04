package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"path"
	"strings"
	"unicode/utf8"

	"gomaccms/internal/user"

	"github.com/gin-gonic/gin"
)

// opLoggedKey 一个请求只记录一次 (部分路由同时套用 AuthTokenInertia 与 AuthToken)
const opLoggedKey = "opLogged"

// mutatingGetSuffixes 以 GET 执行的变更操作 (删除、重试、清理、重新采集)
var mutatingGetSuffixes = []string{"/del", "/retry", "/retry/all", "/clear/done", "/clear/all", "/update/single"}

// verbs 路由最后一段对应的操作名称 (页面级权限的操作日志用)
var verbs = map[string]string{
	"add": "添加", "save": "保存", "update": "修改", "del": "删除", "state": "启用/停用", "change": "启用/停用",
	"basic": "修改设置", "bind": "绑定分类", "transfer": "转移视频", "test": "测试接口",
}

// commonActions 公共路由 (不需要授权) 的操作名称
var commonActions = map[string]string{
	"POST /manage/file/upload":   "上传图片",
	"POST /manage/user/profile":  "个人资料 / 修改资料",
	"POST /manage/user/password": "个人资料 / 修改密码",
}

// isOperation 是否为需要记录操作日志的请求 (非 GET 请求与以 GET 执行的变更操作)
func isOperation(r *http.Request) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
		return true
	}
	for _, s := range mutatingGetSuffixes {
		if strings.HasSuffix(r.URL.Path, s) {
			return true
		}
	}
	return false
}

// DescribeAction 操作日志的操作名称: 权限目录中的 页面 / 操作 名称, 页面级权限再加上路由对应的动作
func DescribeAction(method, p string) string {
	if name, ok := commonActions[method+" "+p]; ok {
		return name
	}
	key, _ := requiredPermission(method, p)
	name, ok := permNames[key]
	if !ok {
		return p
	}
	if actionKeys[key] {
		return name
	}
	if v, ok := verbs[path.Base(p)]; ok {
		return name + " / " + v
	}
	return name + " / " + strings.TrimPrefix(p, "/manage/")
}

// logOperation 执行请求并记录操作日志 (已通过身份验证); denied 为没有权限被拒绝的请求
func logOperation(c *gin.Context, uc *user.UserClaims, denied bool) {
	if !isOperation(c.Request) || c.GetBool(opLoggedKey) {
		if !denied {
			c.Next()
		}
		return
	}
	c.Set(opLoggedKey, true)
	entry := &user.AdminLog{UserId: uc.UserID, UserName: uc.UserName, Ip: c.ClientIP(), Method: c.Request.Method,
		Path: c.Request.URL.Path, Action: DescribeAction(c.Request.Method, c.Request.URL.Path), Params: requestParams(c)}
	if denied {
		entry.Action = "无权限: " + entry.Action
		user.Svc.AddLog(entry)
		return
	}
	rec := &resultRecorder{ResponseWriter: c.Writer}
	c.Writer = rec
	c.Next()
	entry.Success = rec.success(c.GetHeader("X-Inertia") != "")
	user.Svc.AddLog(entry)
}

// requestParams 请求参数: 查询字符串 + JSON / 表单内容, 密码类栏位隐藏, 最长 2000 字符
func requestParams(c *gin.Context) string {
	var parts []string
	if q := c.Request.URL.RawQuery; q != "" {
		parts = append(parts, q)
	}
	ct := c.ContentType()
	if c.Request.Body != nil && (ct == "application/json" || ct == "application/x-www-form-urlencoded") {
		body, _ := io.ReadAll(io.LimitReader(c.Request.Body, 64<<10))
		c.Request.Body = io.NopCloser(bytes.NewReader(body))
		if len(body) > 0 {
			var v any
			if ct == "application/json" && json.Unmarshal(body, &v) == nil {
				body, _ = json.Marshal(redact(v))
			}
			parts = append(parts, string(body))
		}
	} else if strings.HasPrefix(ct, "multipart/") {
		parts = append(parts, "(文件上传)")
	}
	s := strings.Join(parts, " ")
	if utf8.RuneCountInString(s) > 2000 {
		s = string([]rune(s)[:2000]) + "..."
	}
	return s
}

// redact 隐藏 JSON 中名称含 password / pwd 的栏位
func redact(v any) any {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			lk := strings.ToLower(k)
			if strings.Contains(lk, "password") || strings.Contains(lk, "pwd") {
				t[k] = "******"
			} else {
				t[k] = redact(val)
			}
		}
	case []any:
		for i := range t {
			t[i] = redact(t[i])
		}
	}
	return v
}

// resultRecorder 记录响应状态与开头内容, 用来判断操作是否成功
type resultRecorder struct {
	gin.ResponseWriter
	head []byte
}

func (w *resultRecorder) Write(b []byte) (int, error) {
	if len(w.head) < 32 {
		w.head = append(w.head, b[:min(len(b), 32-len(w.head))]...)
	}
	return w.ResponseWriter.Write(b)
}

func (w *resultRecorder) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

// success Inertia 变更请求成功时重定向 (失败时带着错误重新渲染页面); JSON 请求看响应的 code
func (w *resultRecorder) success(isInertia bool) bool {
	status := w.Status()
	if status >= http.StatusBadRequest && status != http.StatusConflict {
		return false
	}
	if isInertia {
		return status != http.StatusOK
	}
	return !bytes.HasPrefix(bytes.TrimSpace(w.head), []byte(`{"code":-1`))
}
