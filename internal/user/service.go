package user

import (
	"errors"
	"gomaccms/internal/config"
	"gomaccms/internal/util"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var Svc *Service

type Service struct {
	userRepo *Repository
}

func NewService(userRepo *Repository) *Service {
	return &Service{userRepo: userRepo}
}

// Login 管理员登录, 成功时记录登录时间、IP 与次数
func (s *Service) Login(account, password, ip string) (token string, err error) {
	if loginLocked(account, ip) {
		return "", errLoginLocked
	}
	u := s.userRepo.FindByNameOrEmail(account)
	if u == nil || util.PasswordEncrypt(password, u.Salt) != u.Password {
		return "", recordLoginFail(account, ip, errors.New("用户名或密码错误"))
	}
	if u.Status != StatusEnabled {
		return "", errors.New("该账户已被停用")
	}
	clearLoginFail(account, ip)
	if token, err = s.GenToken(u.ID, u.UserName); err != nil {
		return "", err
	}
	if err = s.userRepo.SaveToken(token, u.ID); err != nil {
		return "", err
	}
	s.userRepo.RecordLogin(u.ID, ip)
	return token, nil
}

// ChangePassword 修改密码
func (s *Service) ChangePassword(account, password, newPassword string) error {
	u := s.userRepo.FindByNameOrEmail(account)
	if u == nil {
		return errors.New(" 用户信息不存在!!!")
	}
	if util.PasswordEncrypt(password, u.Salt) != u.Password {
		return errors.New("原密码校验失败")
	}
	newUser := User{}
	newUser.ID = u.ID
	newUser.Password = util.PasswordEncrypt(newPassword, u.Salt)
	s.userRepo.UpdateUserInfo(newUser)
	return nil
}

// GetUserInfo 获取用户基本信息
func (s *Service) GetUserInfo(id uint) UserInfoVo {
	u := s.userRepo.FindById(id)
	founder, perms := s.Access(id)
	return UserInfoVo{Id: u.ID, UserName: u.UserName, Email: u.Email, Gender: u.Gender, NickName: u.NickName, Avatar: u.Avatar,
		Status: u.Status, Founder: founder, Permissions: perms}
}

// VerifyUserPassword 校验密码
func (s *Service) VerifyUserPassword(id uint, password string) bool {
	u := s.userRepo.FindById(id)
	return util.PasswordEncrypt(password, u.Salt) == u.Password
}

// GenToken 生成token
func (s *Service) GenToken(userId uint, userName string) (string, error) {
	uc := UserClaims{
		UserID:   userId,
		UserName: userName,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    config.Issuer,
			Subject:   userName,
			Audience:  jwt.ClaimStrings{"Auth_All"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(config.AuthTokenExpires * time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-10 * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        util.GenerateSalt(),
		},
	}
	priKey, err := util.ParsePriKeyBytes([]byte(config.PrivateKey))
	if err != nil {
		return "", err
	}
	return jwt.NewWithClaims(jwt.SigningMethodRS256, uc).SignedString(priKey)
}

// ParseToken 解析token
func (s *Service) ParseToken(tokenStr string) (*UserClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		pub, err := util.ParsePubKeyBytes([]byte(config.PublicKey))
		if err != nil {
			return nil, err
		}
		return pub, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			claims, _ := token.Claims.(*UserClaims)
			return claims, err
		}
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("token is invalid")
	}
	claims, ok := token.Claims.(*UserClaims)
	if !ok {
		return nil, errors.New("invalid claim type error")
	}
	return claims, err
}

// Authenticate 校验 auth-token 字符串;成功返回 (claims, 刷新后的新token或空字符串, "")、
// 失败返回 (nil, "", 失败原因)。刷新逻辑跟原本 middleware 里的完全一致。
func (s *Service) Authenticate(authToken string) (*UserClaims, string, string) {
	uc, err := s.ParseToken(authToken)
	if uc == nil {
		return nil, "", "身份验证信息格式异常,请重新登录!!!"
	}
	t := s.userRepo.GetTokenById(uc.UserID)
	if len(t) <= 0 {
		return nil, "", "身份验证信息已失效,请重新登录!!!"
	}
	if t != authToken {
		return nil, "", "账号在其它设备登录,身份验证信息失效,请重新登录!!!"
	} else if err != nil && errors.Is(err, jwt.ErrTokenExpired) {
		newToken, _ := s.GenToken(uc.UserID, uc.UserName)
		_ = s.userRepo.SaveToken(newToken, uc.UserID)
		uc, _ = s.ParseToken(newToken)
		return uc, newToken, ""
	}
	return uc, "", ""
}

// ClearToken 清除指定用户的登录 token
func (s *Service) ClearToken(userId uint) error {
	return s.userRepo.ClearToken(userId)
}
