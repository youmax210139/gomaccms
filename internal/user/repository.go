package user

import (
	"fmt"
	"gomaccms/internal/config"
	"gomaccms/internal/db"
	"log"
	"time"

	"gorm.io/gorm"
)

// Repo is the package-level User repository instance, assigned once in
// bootstrap.Wire(). It lets background code (spider, bootstrap) reach the
// repository without parameter passing.
var Repo *Repository

type Repository struct{}

func NewRepository() *Repository {
	return &Repository{}
}

// FindByNameOrEmail 查询 username || email 对应的账户信息
func (r *Repository) FindByNameOrEmail(userName string) *User {
	var u *User
	if err := db.Mdb.Where("user_name = ? OR email = ?", userName, userName).First(&u).Error; err != nil {
		log.Println(err)
		return nil
	}
	return u
}

// FindById 通过id获取对应的用户信息
func (r *Repository) FindById(id uint) User {
	var user = User{Model: gorm.Model{ID: id}}
	db.Mdb.First(&user)
	return user
}

// UpdateUserInfo 更新用户信息
func (r *Repository) UpdateUserInfo(u User) {
	db.Mdb.Model(&u).Updates(User{Password: u.Password, Email: u.Email, NickName: u.NickName, Status: u.Status})
}

// SaveToken 将用户登录成功后的token字符串存放到redis中
func (r *Repository) SaveToken(token string, userId uint) error {
	return db.Rdb.Set(db.Cxt, fmt.Sprintf(config.UserTokenKey, userId), token, (config.AuthTokenExpires+7*24)*time.Hour).Err()
}

// GetTokenById 从redis中获取指定userId对应的token
func (r *Repository) GetTokenById(userId uint) string {
	token, err := db.Rdb.Get(db.Cxt, fmt.Sprintf(config.UserTokenKey, userId)).Result()
	if err != nil {
		log.Println("User Token Not Found: ", err)
		return ""
	}
	return token
}

// ClearToken 清除指定id的用户的登录信息
func (r *Repository) ClearToken(userId uint) error {
	return db.Rdb.Del(db.Cxt, fmt.Sprintf(config.UserTokenKey, userId)).Err()
}

// RecordLogin 记录登录时间、IP 并累计登录次数
func (r *Repository) RecordLogin(id uint, ip string) {
	db.Mdb.Model(&User{}).Where("id = ?", id).Updates(map[string]any{
		"last_login_at": time.Now(), "last_login_ip": ip, "login_count": gorm.Expr("login_count + 1"),
	})
}

// ListAdmins 全部管理员
func (r *Repository) ListAdmins() []User {
	var l []User
	if err := db.Mdb.Order("id DESC").Find(&l).Error; err != nil {
		log.Println("List admins failed:", err)
	}
	return l
}

// UserNameTaken 账号名或 Email 是否已被 exceptId 以外的管理员使用
func (r *Repository) UserNameTaken(name string, exceptId uint) bool {
	var count int64
	db.Mdb.Model(&User{}).Where("(user_name = ? OR email = ?) AND id <> ?", name, name, exceptId).Count(&count)
	return count > 0
}

// CountEnabledAdmins 启用中的管理员数量 (除 exceptIds 外)
func (r *Repository) CountEnabledAdmins(exceptIds ...uint) int64 {
	var count int64
	query := db.Mdb.Model(&User{}).Where("status = ?", StatusEnabled)
	if len(exceptIds) > 0 {
		query = query.Where("id NOT IN ?", exceptIds)
	}
	query.Count(&count)
	return count
}

// CreateAdmin 新增管理员
func (r *Repository) CreateAdmin(u *User) error {
	return db.Mdb.Create(u).Error
}

// UpdateAdmin 更新管理员的资料 (columns 指定要更新的栏位, 空字符串也会写入)
func (r *Repository) UpdateAdmin(u *User, columns ...string) error {
	return db.Mdb.Model(&User{}).Where("id = ?", u.ID).Select(columns).Updates(u).Error
}

// DeleteAdmins 删除管理员 (物理删除, 账号名可再次使用)
func (r *Repository) DeleteAdmins(ids []uint) error {
	return db.Mdb.Unscoped().Where("id IN ?", ids).Delete(&User{}).Error
}

// FounderId 创始管理员的ID (最早建立的管理员)
func (r *Repository) FounderId() uint {
	var id uint
	db.Mdb.Model(&User{}).Order("id").Limit(1).Pluck("id", &id)
	return id
}
