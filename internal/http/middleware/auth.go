package middleware

import (
	"gomaccms/internal/config"
	"gomaccms/internal/http/response"
	"gomaccms/internal/http/session"
	"gomaccms/internal/user"
	"gomaccms/internal/view/inertia"
	"net/http"

	"github.com/gin-gonic/gin"
)

// authFlow 后台认证流程中随路由类型而不同的部分: 令牌来源、刷新后的令牌如何下发, 以及未登录 / 无权限时的响应
type authFlow struct {
	token        func(c *gin.Context) string
	refresh      func(c *gin.Context, token string)
	unauthorized func(c *gin.Context, reason string)
	forbidden    func(c *gin.Context)
}

// handler 认证 → 记录身份 → 权限检查 → 执行并记录操作日志
func (f authFlow) handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		authToken := f.token(c)
		if authToken == "" {
			f.unauthorized(c, "用户未授权,请先登录")
			c.Abort()
			return
		}
		uc, refreshedToken, reason := user.Svc.Authenticate(authToken)
		if reason != "" {
			f.unauthorized(c, reason)
			c.Abort()
			return
		}
		if refreshedToken != "" {
			f.refresh(c, refreshedToken)
		}
		session.Set(c, uc)
		if founder, granted := user.Svc.Access(uc.UserID); !HasPermission(c.Request, founder, granted) {
			logOperation(c, uc, true)
			f.forbidden(c)
			c.Abort()
			return
		}
		logOperation(c, uc, false)
	}
}

func cookieToken(c *gin.Context) string {
	t, _ := c.Cookie(config.AuthTokenCookie)
	return t
}

// AuthToken 保护纯 JSON 的后台接口: 读取 auth-token 请求头, 为空时回退读取 auth-token cookie;
// 未登录回 401, 无权限回 403 (JSON), 刷新后的令牌通过 new-token 响应头下发
func AuthToken() gin.HandlerFunc {
	return authFlow{
		token: func(c *gin.Context) string {
			if t := c.Request.Header.Get("auth-token"); t != "" {
				return t
			}
			return cookieToken(c)
		},
		refresh: func(c *gin.Context, token string) { c.Header("new-token", token) },
		unauthorized: func(c *gin.Context, reason string) {
			c.JSON(http.StatusUnauthorized, response.Response{Code: response.SUCCESS, Msg: reason})
		},
		forbidden: func(c *gin.Context) {
			c.JSON(http.StatusForbidden, response.Response{Code: response.FAILED, Msg: "没有权限执行此操作, 请联系创始管理员"})
		},
	}.handler()
}

// AuthTokenInertia 保护 Inertia 渲染的后台页面路由: 只读取 auth-token cookie; 未登录时整页跳转到 /login
// (Inertia 的 XHR 层不会把裸 401 当成导航来处理), 无权限时渲染 Forbidden 页, 刷新后的令牌写回 cookie
func AuthTokenInertia() gin.HandlerFunc {
	return authFlow{
		token:        cookieToken,
		refresh:      SetAuthCookie,
		unauthorized: func(c *gin.Context, _ string) { inertia.I.Location(c.Writer, c.Request, "/login") },
		forbidden:    func(c *gin.Context) { _ = inertia.RenderManage(c, "Forbidden", nil) },
	}.handler()
}

// SetAuthCookie 把 JWT 写入 httpOnly cookie。
func SetAuthCookie(c *gin.Context, token string) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(config.AuthTokenCookie, token, config.AuthTokenExpires*3600, "/", "", c.Request.TLS != nil, true)
}

// ClearAuthCookie 清除 JWT cookie。
func ClearAuthCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(config.AuthTokenCookie, "", -1, "/", "", c.Request.TLS != nil, true)
}

// CurrentUser 返回当前请求的登录用户信息(从 auth-token cookie 读取), 不写任何
// 响应 —— 供 ShowLogin 这类"已登录就跳走"的场景使用。
func CurrentUser(c *gin.Context) (*user.UserClaims, bool) {
	authToken := cookieToken(c)
	if authToken == "" {
		return nil, false
	}
	uc, _, reason := user.Svc.Authenticate(authToken)
	return uc, reason == ""
}
