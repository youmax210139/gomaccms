package admin

import (
	"fmt"
	"gomaccms/internal/user"
	"gomaccms/internal/view/inertia"
	"net/url"

	"github.com/gin-gonic/gin"
	gonertia "github.com/romsar/gonertia/v3"
)

// AdminLogList 操作日志页(Inertia 渲染)
func AdminLogList(c *gin.Context) {
	q := user.LogQuery{UserName: c.Query("userName"), Keyword: c.Query("keyword"), Success: c.Query("success"), Paging: pageFrom(c.Request.URL.Query())}
	props := gonertia.Props{
		"logs":      user.Svc.ListLogs(q),
		"userNames": user.Svc.LogUserNames(),
		"query":     q,
	}
	if msg := c.Query("msg"); msg != "" {
		props["flash"] = msg
	}
	_ = inertia.RenderManage(c, "User/AdminLogs", props)
}

// AdminLogClear 清理操作日志: days 天前的日志, 0 为全部(Inertia 渲染)
func AdminLogClear(c *gin.Context) {
	var req struct {
		Days int `json:"days"`
	}
	_ = c.ShouldBindJSON(&req)
	n, err := user.Svc.ClearLogs(max(req.Days, 0))
	msg := fmt.Sprintf("已清理 %d 条日志", n)
	if err != nil {
		msg = "清理失败: " + err.Error()
	}
	inertia.Redirect(c, "/manage/admin/log/list?msg="+url.QueryEscape(msg))
}
