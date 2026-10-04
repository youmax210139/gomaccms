package user

// UserInfoVo 用户信息返回对象
type UserInfoVo struct {
	Id       uint   `json:"id"`
	UserName string `json:"userName"`
	Email    string `json:"email"`
	Gender   int    `json:"gender"`
	NickName string `json:"nickName"`
	Avatar   string `json:"avatar"`
	Status   int    `json:"status"`
	// Founder 创始管理员 (拥有全部权限); Permissions 其他管理员的后台权限 key
	Founder     bool     `json:"founder"`
	Permissions []string `json:"permissions"`
}
