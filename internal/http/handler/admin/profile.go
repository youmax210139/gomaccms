package admin

import (
	"gomaccms/internal/http/session"
	"gomaccms/internal/user"
	"gomaccms/internal/view/inertia"

	"github.com/gin-gonic/gin"
	gonertia "github.com/romsar/gonertia/v3"
)

// renderProfile 渲染个人资料页; errMsg 非空时作为错误提示
func renderProfile(c *gin.Context, errMsg string) {
	props := gonertia.Props{}
	withFormError(props, errMsg)
	_ = inertia.RenderManage(c, "User/Profile", props)
}

// Profile 个人资料页(Inertia 渲染)
func Profile(c *gin.Context) {
	renderProfile(c, "")
}

// ProfileSave 修改自己的昵称 / Email / 头像(Inertia 渲染)
func ProfileSave(c *gin.Context) {
	var req struct {
		NickName string `json:"nickName"`
		Email    string `json:"email"`
		Avatar   string `json:"avatar"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		renderProfile(c, "请求参数异常")
		return
	}
	if err := user.Svc.UpdateProfile(session.UserID(c), req.NickName, req.Email, req.Avatar); err != nil {
		renderProfile(c, err.Error())
		return
	}
	inertia.Redirect(c, "/manage/user/profile")
}

// ProfilePassword 修改自己的密码(Inertia 渲染)
func ProfilePassword(c *gin.Context) {
	var req struct {
		Password    string `json:"password"`
		NewPassword string `json:"newPassword"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		renderProfile(c, "请求参数异常")
		return
	}
	if err := user.Svc.ChangeOwnPassword(session.UserID(c), req.Password, req.NewPassword); err != nil {
		renderProfile(c, err.Error())
		return
	}
	inertia.Redirect(c, "/manage/user/profile")
}
