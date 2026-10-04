package member

import (
	"errors"
	"gomaccms/internal/util"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

var Svc *Service

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// GroupVo 会员组列表项 (带会员数)
type GroupVo struct {
	Group
	Builtin bool  `json:"builtin"`
	Members int64 `json:"members"`
}

// ListGroups 会员组列表
func (s *Service) ListGroups() []GroupVo {
	counts := s.repo.GroupMemberCounts()
	var l []GroupVo
	for _, g := range s.repo.ListGroups() {
		l = append(l, GroupVo{Group: g, Builtin: g.Builtin(), Members: counts[g.Id]})
	}
	return l
}

// SaveGroup 新增 (Id 为 0) 或修改会员组; 内置会员组不能停用
func (s *Service) SaveGroup(g Group) error {
	g.Name, g.Remark = strings.TrimSpace(g.Name), strings.TrimSpace(g.Remark)
	switch {
	case g.Name == "" || utf8.RuneCountInString(g.Name) > 30:
		return errors.New("名称不能为空且不能超过 30 个字符")
	case utf8.RuneCountInString(g.Remark) > 255:
		return errors.New("备注不能超过 255 个字符")
	case g.PriceDay < 0 || g.PriceWeek < 0 || g.PriceMonth < 0 || g.PriceYear < 0:
		return errors.New("价格不能小于 0")
	}
	if g.Builtin() {
		g.Status = true
	}
	g.Permissions = cleanPermissions(g.Permissions)
	if g.Id == 0 {
		return s.repo.CreateGroup(&g)
	}
	if _, ok := s.repo.FindGroup(g.Id); !ok {
		return errors.New("会员组不存在")
	}
	return s.repo.UpdateGroup(&g, "updated_at", "name", "status", "price_day", "price_week", "price_month", "price_year", "remark", "permissions")
}

// cleanPermissions 去掉无效的权限值
func cleanPermissions(perms map[int64][]string) map[int64][]string {
	out := make(map[int64][]string, len(perms))
	for cid, l := range perms {
		var valid []string
		for _, p := range AllPerms {
			if slices.Contains(l, p) {
				valid = append(valid, p)
			}
		}
		if valid == nil {
			valid = []string{}
		}
		out[cid] = valid
	}
	return out
}

// SetGroupStatus 启用 / 停用会员组 (内置会员组不能停用)
func (s *Service) SetGroupStatus(id int64, status bool) error {
	g, ok := s.repo.FindGroup(id)
	if !ok {
		return errors.New("会员组不存在")
	}
	if g.Builtin() && !status {
		return errors.New("内置会员组不能停用")
	}
	g.Status = status
	return s.repo.UpdateGroup(&g, "updated_at", "status")
}

// DeleteGroups 删除会员组 (内置会员组不能删除), 其会员改为默认会员
func (s *Service) DeleteGroups(ids []int64) error {
	if len(ids) == 0 {
		return errors.New("请先选择会员组")
	}
	if slices.Contains(ids, GuestGroupId) || slices.Contains(ids, DefaultGroupId) {
		return errors.New("内置会员组不能删除")
	}
	return s.repo.DeleteGroups(ids)
}

// MemberInput 新增 (Id 为 0) / 修改会员; 修改时 Password 为空表示不修改密码
type MemberInput struct {
	Id       int64  `json:"id"`
	UserName string `json:"userName"`
	Password string `json:"password"`
	NickName string `json:"nickName"`
	Email    string `json:"email"`
	GroupId  int64  `json:"groupId"`
	Points   int    `json:"points"`
	ExpireAt string `json:"expireAt"` // yyyy-MM-dd HH:mm:ss, 空为不过期
	Status   bool   `json:"status"`
}

// ListMembers 会员列表
func (s *Service) ListMembers(q MemberQuery) []Member {
	return s.repo.ListMembers(q)
}

// SaveMember 新增或修改会员
func (s *Service) SaveMember(in MemberInput) error {
	in.UserName, in.NickName, in.Email = strings.TrimSpace(in.UserName), strings.TrimSpace(in.NickName), strings.TrimSpace(in.Email)
	switch {
	case !util.ValidAccount(in.UserName):
		return errors.New("账号只能包含字母、数字与下划线, 长度 3-30")
	case utf8.RuneCountInString(in.NickName) > 60:
		return errors.New("昵称不能超过 60 个字符")
	case in.Email != "" && !util.ValidEmail(in.Email):
		return errors.New("Email 格式错误")
	case s.repo.UserNameTaken(in.UserName, in.Id):
		return errors.New("账号已被其他会员使用")
	case in.Id == 0 && len(in.Password) < 6, in.Id != 0 && in.Password != "" && len(in.Password) < 6:
		return errors.New("密码至少 6 个字符")
	case in.Points < 0:
		return errors.New("积分不能小于 0")
	case in.GroupId == GuestGroupId:
		return errors.New("会员不能属于游客组")
	}
	if _, ok := s.repo.FindGroup(in.GroupId); !ok {
		return errors.New("会员组不存在")
	}
	var expireAt *time.Time
	if in.ExpireAt != "" {
		t, err := time.ParseInLocation(time.DateTime, in.ExpireAt, time.Local)
		if err != nil {
			return errors.New("到期时间格式错误")
		}
		expireAt = &t
	}
	if in.NickName == "" {
		in.NickName = in.UserName
	}
	m := &Member{Id: in.Id, UserName: in.UserName, NickName: in.NickName, Email: in.Email, GroupId: in.GroupId,
		Points: in.Points, ExpireAt: expireAt, Status: in.Status}
	if in.Id == 0 {
		m.Salt = util.GenerateSalt()
		m.Password = util.PasswordEncrypt(in.Password, m.Salt)
		return s.repo.CreateMember(m)
	}
	old, ok := s.repo.FindMember(in.Id)
	if !ok {
		return errors.New("会员不存在")
	}
	columns := []string{"updated_at", "user_name", "nick_name", "email", "group_id", "points", "expire_at", "status"}
	if in.Password != "" {
		m.Password = util.PasswordEncrypt(in.Password, old.Salt)
		columns = append(columns, "password")
	}
	return s.repo.UpdateMember(m, columns...)
}

// SetMemberStatus 启用 / 停用会员
func (s *Service) SetMemberStatus(id int64, status bool) error {
	return s.repo.UpdateMember(&Member{Id: id, Status: status}, "updated_at", "status")
}

// DeleteMembers 删除会员
func (s *Service) DeleteMembers(ids []int64) error {
	if len(ids) == 0 {
		return errors.New("请先选择会员")
	}
	return s.repo.DeleteMembers(ids)
}

// GuestGroup 游客组 (前台访客的权限); 查询失败时返回允许全部的空权限
func (s *Service) GuestGroup() Group {
	g, _ := s.repo.FindGroup(GuestGroupId)
	return g
}

// GuestAllows 游客对分类 (视频的 cid 与 pid 都要允许) 是否有某个权限
func (s *Service) GuestAllows(perm string, categoryIds ...int64) bool {
	g := s.GuestGroup()
	for _, id := range categoryIds {
		if id > 0 && !g.Allows(id, perm) {
			return false
		}
	}
	return true
}
