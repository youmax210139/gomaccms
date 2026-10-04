package admin

import (
	"gomaccms/internal/http/response"
	"gomaccms/internal/siteconfig"

	"github.com/gin-gonic/gin"
	"github.com/mojocn/base64Captcha"
)

// loginCaptcha 后台登录图形验证码: 4 位数字, 答案存于进程内存 (默认 10 分钟过期, 验证一次即失效)
var loginCaptcha = base64Captcha.NewCaptcha(
	base64Captcha.NewDriverDigit(48, 140, 4, 0.5, 60),
	base64Captcha.DefaultMemStore,
)

// captchaEnabled 后台「登录验证码」是否开启
func captchaEnabled() bool {
	return siteconfig.Svc.GetSiteBasicConfig().LoginCaptcha
}

// verifyLoginCaptcha 校验登录验证码 (未开启时直接通过)
func verifyLoginCaptcha(id, answer string) bool {
	if !captchaEnabled() {
		return true
	}
	return id != "" && answer != "" && loginCaptcha.Verify(id, answer, true)
}

// LoginCaptcha 生成一张登录验证码图片 (base64), 未开启验证码时返回失败
func LoginCaptcha(c *gin.Context) {
	if !captchaEnabled() {
		response.Failed("未开启登录验证码", c)
		return
	}
	id, b64, _, err := loginCaptcha.Generate()
	if err != nil {
		response.Failed("验证码生成失败", c)
		return
	}
	response.Success(gin.H{"id": id, "image": b64}, "验证码获取成功", c)
}
