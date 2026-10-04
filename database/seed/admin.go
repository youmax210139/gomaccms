package seed

import (
	"gomaccms/internal/db"
	"gomaccms/internal/user"
	"gomaccms/internal/util"
)

// seedAdmin 初始化 admin 账户 (原 user.Repo.InitAdminAccount):
// 已存在 admin 时直接跳过, 不会覆盖其密码
func seedAdmin() error {
	if user.Repo.FindByNameOrEmail("admin") != nil {
		return nil
	}
	u := &user.User{
		UserName: "admin",
		Password: "admin",
		Salt:     util.GenerateSalt(),
		Email:    "administrator@gmail.com",
		Gender:   2,
		NickName: "admin",
		Avatar:   "empty",
		Status:   user.StatusEnabled,
	}
	u.Password = util.PasswordEncrypt(u.Password, u.Salt)
	return db.Mdb.Create(u).Error
}
