package seed

import (
	"errors"

	"gomaccms/internal/db"
	"gomaccms/internal/user"
	"gomaccms/internal/util"

	"gorm.io/gorm"
)

// seedAdmin 还没有任何管理员时 (例如直接执行 cmd/migrate seed) 建立预设账号 admin/admin;
// 已有管理员 (安装向导建立的或旧数据) 时跳过
func seedAdmin() error {
	var n int64
	if err := db.Mdb.Model(&user.User{}).Count(&n).Error; err != nil || n > 0 {
		return err
	}
	return CreateAdmin("admin", "administrator@gmail.com", "admin")
}

// CreateAdmin 建立启用状态的管理员 (安装向导与 seedAdmin 共用);
// 同名账号已存在时更新其 Email 与密码 (安装向导失败后重试的情况)
func CreateAdmin(account, email, password string) error {
	var u user.User
	err := db.Mdb.Where("user_name = ?", account).First(&u).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if u.ID == 0 {
		u.UserName, u.NickName, u.Gender, u.Avatar = account, account, 2, "empty"
	}
	u.Email, u.Status = email, user.StatusEnabled
	u.Salt = util.GenerateSalt()
	u.Password = util.PasswordEncrypt(password, u.Salt)
	return db.Mdb.Save(&u).Error
}
