package user

import (
	"errors"
	"gomaccms/internal/util"
	"strings"
	"unicode/utf8"
)

// UpdateProfile 修改自己的昵称、Email 与头像
func (s *Service) UpdateProfile(id uint, nickName, email, avatar string) error {
	nickName, email, avatar = strings.TrimSpace(nickName), strings.TrimSpace(email), strings.TrimSpace(avatar)
	switch {
	case nickName == "" || utf8.RuneCountInString(nickName) > 60:
		return errors.New("昵称不能为空且不能超过 60 个字符")
	case email != "" && !util.ValidEmail(email):
		return errors.New("Email 格式错误")
	case email != "" && s.userRepo.UserNameTaken(email, id):
		return errors.New("Email 已被其他管理员使用")
	case len(avatar) > 255:
		return errors.New("头像地址过长")
	}
	if avatar == "" {
		avatar = "empty"
	}
	u := &User{NickName: nickName, Email: email, Avatar: avatar}
	u.ID = id
	return s.userRepo.UpdateAdmin(u, "updated_at", "nick_name", "email", "avatar")
}

// ChangeOwnPassword 修改自己的密码 (需验证原密码)
func (s *Service) ChangeOwnPassword(id uint, oldPassword, newPassword string) error {
	u := s.userRepo.FindById(id)
	switch {
	case u.ID == 0:
		return errors.New("管理员不存在")
	case util.PasswordEncrypt(oldPassword, u.Salt) != u.Password:
		return errors.New("原密码错误")
	case len(newPassword) < 6:
		return errors.New("新密码至少 6 个字符")
	case newPassword == oldPassword:
		return errors.New("新密码不能与原密码相同")
	}
	n := &User{Password: util.PasswordEncrypt(newPassword, u.Salt)}
	n.ID = id
	return s.userRepo.UpdateAdmin(n, "updated_at", "password")
}
