package admin

import (
	"fmt"
	"gomaccms/internal/http/middleware"
	"gomaccms/internal/http/request"
	"gomaccms/internal/http/response"
	"gomaccms/internal/http/session"
	"gomaccms/internal/siteconfig"
	"gomaccms/internal/user"
	"gomaccms/internal/util"
	"gomaccms/internal/view/inertia"

	"github.com/gin-gonic/gin"
	gonertia "github.com/romsar/gonertia/v3"
)

// ShowLogin 渲染登录页; 已登录则直接跳转到后台首页
func ShowLogin(c *gin.Context) {
	if _, ok := middleware.CurrentUser(c); ok {
		inertia.Redirect(c, "/manage/index")
		return
	}
	renderLogin(c, "")
}

// renderLogin 渲染登录页; errMsg 非空时作为错误提示
func renderLogin(c *gin.Context, errMsg string) {
	props := gonertia.Props{"captcha": captchaEnabled(), "siteName": siteconfig.Svc.GetSiteBasicConfig().SiteName}
	withFormError(props, errMsg)
	_ = inertia.I.Render(c.Writer, c.Request, "Login", props)
}

// Login 管理员登录接口(Inertia)
func Login(c *gin.Context) {
	var req request.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.UserName) <= 0 || len(req.Password) <= 0 {
		renderLogin(c, "用户名和密码信息不能为空")
		return
	}
	// 验证码先于密码校验, 且验证一次即失效, 避免借登录接口暴力尝试密码
	if !verifyLoginCaptcha(req.CaptchaId, req.Captcha) {
		renderLogin(c, "验证码错误或已过期")
		return
	}
	token, err := user.Svc.Login(req.UserName, req.Password, c.ClientIP())
	user.Svc.LogLogin(req.UserName, c.ClientIP(), err)
	if err != nil {
		renderLogin(c, err.Error())
		return
	}
	middleware.SetAuthCookie(c, token)
	inertia.Redirect(c, "/manage/index")
}

// Logout 退出登录
func Logout(c *gin.Context) {
	uc, ok := session.Claims(c)
	if !ok {
		response.Failed("请求失败,登录信息获取异常!!!", c)
		return
	}
	user.Svc.AddLog(&user.AdminLog{UserId: uc.UserID, UserName: uc.UserName, Ip: c.ClientIP(), Method: c.Request.Method,
		Path: c.Request.URL.Path, Action: user.ActionLogout, Success: true})
	err := user.Svc.ClearToken(uc.UserID)
	if err != nil {
		fmt.Println("user logOut err: ", err)
	}
	middleware.ClearAuthCookie(c)
	response.SuccessOnlyMsg("已退出登录!!!", c)
}

// UserPasswordChange 修改用户密码
func UserPasswordChange(c *gin.Context) {
	var req request.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Failed("参数校验失败!!!", c)
		return
	}
	if req.Password == "" || req.NewPassword == "" {
		response.Failed("密码不能为空!!!", c)
		return
	}
	if err := util.ValidPwd(req.NewPassword); err != nil {
		response.Failed(fmt.Sprint("密码格式校验失败: ", err.Error()), c)
		return
	}
	uc, ok := session.Claims(c)
	if !ok {
		response.Failed("操作失败,登录信息异常!!!", c)
		return
	}
	if err := user.Svc.ChangePassword(uc.UserName, req.Password, req.NewPassword); err != nil {
		response.Failed(fmt.Sprint("密码修改失败: ", err.Error()), c)
		return
	}
	response.SuccessOnlyMsg("密码修改成功", c)
}

// UserInfo 获取当前登录用户信息
func UserInfo(c *gin.Context) {
	uc, ok := session.Claims(c)
	if !ok {
		response.Failed("用户信息获取失败, 未获取到用户授权信息", c)
		return
	}
	info := user.Svc.GetUserInfo(uc.UserID)
	response.Success(info, "成功获取用户信息", c)
}
