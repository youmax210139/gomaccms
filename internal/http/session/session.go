// Package session 在 gin.Context 中存取当前登录的管理员 (由认证中间件写入)。
package session

import (
	"gomaccms/internal/config"
	"gomaccms/internal/user"

	"github.com/gin-gonic/gin"
)

// Set 记录当前请求的管理员身份
func Set(c *gin.Context, uc *user.UserClaims) {
	c.Set(config.AuthUserClaims, uc)
}

// Claims 当前请求的管理员身份 (未经认证中间件的请求返回 false)
func Claims(c *gin.Context) (*user.UserClaims, bool) {
	v, ok := c.Get(config.AuthUserClaims)
	if !ok {
		return nil, false
	}
	uc, ok := v.(*user.UserClaims)
	return uc, ok
}

// UserID 当前管理员ID, 未登录时为 0
func UserID(c *gin.Context) uint {
	if uc, ok := Claims(c); ok {
		return uc.UserID
	}
	return 0
}
