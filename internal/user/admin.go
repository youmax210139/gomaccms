package user

import (
	"errors"
	"gomaccms/internal/util"
	"slices"
	"strings"
	"unicode/utf8"
)

// AdminInput 新增 (Id 为 0) / 修改管理员; 修改时 Password 为空表示不修改密码
type AdminInput struct {
	Id       uint
	UserName string
	Password string
	NickName string
	Email    string
	Status   int
	// Permissions 后台权限 key (调用方需先过滤为有效的 key); 创始管理员忽略
	Permissions []string
}

// ListAdmins 全部管理员 (不含密码)
func (s *Service) ListAdmins() []User {
	l := s.userRepo.ListAdmins()
	for i := range l {
		l[i].Password, l[i].Salt = "", ""
	}
	return l
}

// SaveAdmin 新增或修改管理员; currentId 为当前登录的管理员 (不能停用自己)
func (s *Service) SaveAdmin(in AdminInput, currentId uint) error {
	in.UserName, in.NickName, in.Email = strings.TrimSpace(in.UserName), strings.TrimSpace(in.NickName), strings.TrimSpace(in.Email)
	switch {
	case !util.ValidAccount(in.UserName):
		return errors.New("账号只能包含字母、数字与下划线, 长度 3-30")
	case utf8.RuneCountInString(in.NickName) > 60:
		return errors.New("昵称不能超过 60 个字符")
	case in.Email != "" && !util.ValidEmail(in.Email):
		return errors.New("Email 格式错误")
	case s.userRepo.UserNameTaken(in.UserName, in.Id) || (in.Email != "" && s.userRepo.UserNameTaken(in.Email, in.Id)):
		return errors.New("账号或 Email 已被其他管理员使用")
	case in.Id == 0 && len(in.Password) < 6, in.Id != 0 && in.Password != "" && len(in.Password) < 6:
		return errors.New("密码至少 6 个字符")
	case in.Status != StatusEnabled && in.Status != StatusDisabled:
		return errors.New("状态异常")
	}
	if in.NickName == "" {
		in.NickName = in.UserName
	}
	if in.Id == 0 {
		u := &User{UserName: in.UserName, NickName: in.NickName, Email: in.Email, Status: in.Status, Avatar: "empty",
			Salt: util.GenerateSalt(), Permissions: in.Permissions}
		u.Password = util.PasswordEncrypt(in.Password, u.Salt)
		return s.userRepo.CreateAdmin(u)
	}
	old := s.userRepo.FindById(in.Id)
	if old.ID == 0 {
		return errors.New("管理员不存在")
	}
	if in.Status == StatusDisabled {
		if err := s.checkDisable([]uint{in.Id}, currentId); err != nil {
			return err
		}
	}
	u := &User{UserName: in.UserName, NickName: in.NickName, Email: in.Email, Status: in.Status, Permissions: in.Permissions}
	u.ID = in.Id
	columns := []string{"updated_at", "user_name", "nick_name", "email", "status"}
	if in.Id != s.userRepo.FounderId() {
		columns = append(columns, "permissions")
	}
	if in.Password != "" {
		u.Password = util.PasswordEncrypt(in.Password, old.Salt)
		columns = append(columns, "password")
	}
	if err := s.userRepo.UpdateAdmin(u, columns...); err != nil {
		return err
	}
	// 停用或修改密码后需要重新登录
	if in.Status == StatusDisabled || in.Password != "" {
		_ = s.userRepo.ClearToken(in.Id)
	}
	return nil
}

// SetAdminStatus 启用 / 停用管理员 (停用后立即失去登录状态)
func (s *Service) SetAdminStatus(id uint, status int, currentId uint) error {
	if status != StatusEnabled && status != StatusDisabled {
		return errors.New("状态异常")
	}
	if status == StatusDisabled {
		if err := s.checkDisable([]uint{id}, currentId); err != nil {
			return err
		}
	}
	u := &User{Status: status}
	u.ID = id
	if err := s.userRepo.UpdateAdmin(u, "updated_at", "status"); err != nil {
		return err
	}
	if status == StatusDisabled {
		_ = s.userRepo.ClearToken(id)
	}
	return nil
}

// DeleteAdmins 删除管理员 (不能删除自己, 至少保留一个启用中的管理员)
func (s *Service) DeleteAdmins(ids []uint, currentId uint) error {
	if len(ids) == 0 {
		return errors.New("请先选择管理员")
	}
	if err := s.checkDisable(ids, currentId); err != nil {
		return err
	}
	if err := s.userRepo.DeleteAdmins(ids); err != nil {
		return err
	}
	for _, id := range ids {
		_ = s.userRepo.ClearToken(id)
	}
	return nil
}

// checkDisable 停用 / 删除前的检查: 不能是当前登录的管理员, 且之后仍有启用中的管理员
func (s *Service) checkDisable(ids []uint, currentId uint) error {
	if slices.Contains(ids, currentId) {
		return errors.New("不能停用或删除当前登录的管理员")
	}
	if slices.Contains(ids, s.userRepo.FounderId()) {
		return errors.New("不能停用或删除创始管理员")
	}
	if s.userRepo.CountEnabledAdmins(ids...) == 0 {
		return errors.New("至少需要保留一个启用中的管理员")
	}
	return nil
}

// Access 管理员的权限: 是否为创始管理员 (拥有全部权限), 以及其他管理员被授予的权限 key
func (s *Service) Access(id uint) (founder bool, permissions []string) {
	if id == s.userRepo.FounderId() {
		return true, nil
	}
	return false, s.userRepo.FindById(id).Permissions
}
