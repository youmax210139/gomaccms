package user

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	UserName string `json:"userName"`
	Password string `json:"password"`
	Salt     string `json:"salt"`
	Email    string `json:"email"`
	Gender   int    `json:"gender"`
	NickName string `json:"nickName"`
	Avatar   string `json:"avatar"`
	Status   int    `json:"status"` // 1 启用 0 停用 (停用后不能登录)
	Reserve1 string `json:"reserve1"`
	Reserve2 string `json:"reserve2"`
	Reserve3 string `json:"reserve3"`
	// 登录统计
	LastLoginAt *time.Time `json:"lastLoginAt"`
	LastLoginIp string     `json:"lastLoginIp"`
	LoginCount  int        `json:"loginCount"`
	// Permissions 后台权限 key (见 middleware.Permissions); 创始管理员拥有全部权限, 不看此栏位
	Permissions []string `json:"permissions" gorm:"serializer:json"`
}

// 管理员状态
const (
	StatusDisabled = 0
	StatusEnabled  = 1
)

// TableName 设置user表的表名
func (u *User) TableName() string {
	return "users"
}

type UserClaims struct {
	UserID   uint   `json:"userID"`
	UserName string `json:"userName"`
	jwt.RegisteredClaims
}
