package inertia

import (
	"gomaccms/internal/http/session"
	"gomaccms/internal/siteconfig"
	"gomaccms/internal/user"

	"github.com/gin-gonic/gin"
	gonertia "github.com/romsar/gonertia/v3"
)

// RenderManage 渲染一个已登录的 /manage/* Inertia 页面, 自动带上目前网站基本
// 设置、登录用户信息, 页面组件不用再各自打 API 拿这些数据。
func RenderManage(c *gin.Context, component string, props gonertia.Props) error {
	if props == nil {
		props = gonertia.Props{}
	}
	props["site"] = siteconfig.Svc.GetSiteBasicConfig()

	if uc, ok := session.Claims(c); ok {
		props["currentUser"] = user.Svc.GetUserInfo(uc.UserID)
	}

	return I.Render(c.Writer, c.Request, component, props)
}

// Redirect Inertia 重定向 (操作成功后回到列表页等)
func Redirect(c *gin.Context, url string) {
	I.Redirect(c.Writer, c.Request, url)
}

// Back Inertia 重定向回上一页 (保留来源页的查询参数)
func Back(c *gin.Context) {
	I.Back(c.Writer, c.Request)
}
