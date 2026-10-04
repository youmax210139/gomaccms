package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Result 构建response返回数据结构
func Result(code int, data any, msg string, c *gin.Context) {
	c.JSON(http.StatusOK, Response{
		Code: code,
		Data: data,
		Msg:  msg,
	})
}

// Success 成功响应 数据 + 成功提示
func Success(data any, message string, c *gin.Context) {
	Result(SUCCESS, data, message, c)
}

// SuccessOnlyMsg 成功响应, 只返回成功信息
func SuccessOnlyMsg(message string, c *gin.Context) {
	Result(SUCCESS, nil, message, c)
}

// Failed 响应失败 只返回错误信息
func Failed(message string, c *gin.Context) {
	Result(FAILED, nil, message, c)
}
