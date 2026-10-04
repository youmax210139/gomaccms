package middleware

import (
	"net/http"
	"slices"
	"strings"
)

// Permission 后台权限 (参照苹果 CMS): 每个菜单页面一个权限, Actions 为页面内的操作按钮 (显示为 --开头);
// Paths 为该权限对应的后台路由, 按最长前缀匹配, "METHOD path" 只匹配该请求方法
type Permission struct {
	Key     string       `json:"key"`
	Name    string       `json:"name"`
	Paths   []string     `json:"-"`
	Actions []Permission `json:"actions,omitempty"`
}

// PermissionGroup 一组权限 (对应侧边栏的一个菜单组)
type PermissionGroup struct {
	Name  string       `json:"name"`
	Items []Permission `json:"items"`
}

// Permissions 后台权限目录; 修改后台路由时同步更新, 未列出的 /manage 路由除 commonPaths 外一律拒绝
var Permissions = []PermissionGroup{
	{Name: "系统", Items: []Permission{
		{Key: "config", Name: "站群管理", Paths: []string{"/manage/config/"}},
		{Key: "banner", Name: "海报管理", Paths: []string{"/manage/banner/"}},
		{Key: "publish", Name: "网站地图", Paths: []string{"/manage/publish/"}},
		{Key: "cache", Name: "缓存管理", Paths: []string{"/manage/cache/"}},
	}},
	{Name: "基础", Items: []Permission{
		{Key: "category", Name: "分类管理", Paths: []string{"/manage/film/class/"}, Actions: []Permission{
			{Key: "category.scheme", Name: "方案管理", Paths: []string{"/manage/film/class/scheme/"}},
		}},
		{Key: "ad", Name: "广告管理", Paths: []string{"/manage/ad/"}},
		{Key: "lang", Name: "语言管理", Paths: []string{"/manage/lang/"}},
	}},
	{Name: "视频", Items: []Permission{
		{Key: "player", Name: "播放器", Paths: []string{"/manage/film/player"}},
		{Key: "vod", Name: "视频数据", Paths: []string{"/manage/film/search/list"}, Actions: []Permission{
			{Key: "vod.add", Name: "添加视频", Paths: []string{"/manage/film/add"}},
			{Key: "vod.edit", Name: "编辑视频", Paths: []string{"/manage/film/edit"}},
			{Key: "vod.delete", Name: "删除视频", Paths: []string{"/manage/film/search/del", "/manage/film/search/batch/del"}},
			{Key: "vod.batch", Name: "批量设置", Paths: []string{"/manage/film/search/batch/update"}},
			{Key: "vod.recollect", Name: "重新采集", Paths: []string{"/manage/spider/update/single"}},
		}},
	}},
	{Name: "采集", Items: []Permission{
		{Key: "collect", Name: "采集接口", Paths: []string{"/manage/collect/"}, Actions: []Permission{
			{Key: "collect.run", Name: "执行采集", Paths: []string{"/manage/spider/start", "/manage/spider/stop", "/manage/spider/resume", "/manage/spider/retry", "/manage/collect/browse/collect"}},
			{Key: "collect.clear", Name: "清空视频", Paths: []string{"/manage/collect/clear"}},
		}},
	}},
	{Name: "定时任务", Items: []Permission{
		{Key: "cron", Name: "任务管理", Paths: []string{"/manage/cron/"}},
	}},
	{Name: "文件管理", Items: []Permission{
		{Key: "gallery", Name: "图库管理", Paths: []string{"GET /manage/file/upload", "/manage/file/upload/multiple", "/manage/file/del", "/manage/file/gallery"}},
	}},
	{Name: "用户", Items: []Permission{
		{Key: "admin", Name: "管理员", Paths: []string{"/manage/admin/"}},
		{Key: "member.group", Name: "会员组", Paths: []string{"/manage/member/group/"}},
		{Key: "member", Name: "会员", Paths: []string{"/manage/member/"}},
		{Key: "admin.log", Name: "操作日志", Paths: []string{"/manage/admin/log/"}},
	}},
}

// commonPaths 所有管理员都可使用的后台路由 (首页、个人信息, 以及多个页面共用的上传、选图、采集进度等)
var commonPaths = []string{
	"/manage/index", "/manage/user/", "POST /manage/file/upload", "/manage/file/list",
	"/manage/collect/progress", "/manage/collect/options",
}

type permRule struct {
	method, path, key string
}

var permRules = func() []permRule {
	var rules []permRule
	add := func(key string, paths []string) {
		for _, p := range paths {
			method, path := "", p
			if m, rest, ok := strings.Cut(p, " "); ok {
				method, path = m, rest
			}
			rules = append(rules, permRule{method, path, key})
		}
	}
	for _, g := range Permissions {
		for _, item := range g.Items {
			add(item.Key, item.Paths)
			permNames[item.Key] = item.Name
			for _, a := range item.Actions {
				add(a.Key, a.Paths)
				permNames[a.Key] = item.Name + " / " + a.Name
				actionKeys[a.Key] = true
			}
		}
	}
	add("", commonPaths)
	return rules
}()

// permNames 权限 key → 「页面 / 操作」名称 (操作日志用)
var permNames = map[string]string{}

// actionKeys 页面内操作 (--开头) 的权限 key
var actionKeys = map[string]bool{}

// requiredPermission 请求需要的权限 key: 最长前缀匹配; 公共路由返回 ("", true), 未登记的路由返回 ("", false)
func requiredPermission(method, path string) (string, bool) {
	best := -1
	key := ""
	for _, r := range permRules {
		if r.method != "" && r.method != method {
			continue
		}
		if path == r.path || (strings.HasPrefix(path, r.path) && (strings.HasSuffix(r.path, "/") || strings.HasPrefix(path[len(r.path):], "/"))) {
			if len(r.path) > best {
				best, key = len(r.path), r.key
			}
		}
	}
	return key, best >= 0
}

// HasPermission 是否有权限执行该请求 (创始管理员拥有全部权限)
func HasPermission(r *http.Request, founder bool, granted []string) bool {
	// 权限目录只管 /manage 后台路由; /logout、/changePassword 等账号自身的操作只需登录
	if founder || !strings.HasPrefix(r.URL.Path, "/manage/") {
		return true
	}
	key, known := requiredPermission(r.Method, r.URL.Path)
	if !known {
		return false
	}
	return key == "" || slices.Contains(granted, key)
}

// AllPermissionKeys 全部权限 key (校验管理员权限设置用)
func AllPermissionKeys() []string {
	var keys []string
	for _, g := range Permissions {
		for _, item := range g.Items {
			keys = append(keys, item.Key)
			for _, a := range item.Actions {
				keys = append(keys, a.Key)
			}
		}
	}
	return keys
}
