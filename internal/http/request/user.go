package request

// LoginRequest 登录请求参数
type LoginRequest struct {
	UserName string `json:"userName"`
	Password string `json:"password"`
	// 后台开启登录验证码时必填
	CaptchaId string `json:"captchaId"`
	Captcha   string `json:"captcha"`
}

// ChangePasswordRequest 密码修改请求参数
type ChangePasswordRequest struct {
	Password    string `json:"password"`
	NewPassword string `json:"newPassword"`
}
