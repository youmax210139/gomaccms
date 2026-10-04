package member

import "time"

// 内置会员组: 游客 (未登录的访客, 前台目前没有会员登录, 所有访客都套用游客的权限) 与默认会员 (新会员与会员组到期后的会员组)
const (
	GuestGroupId   int64 = 1
	DefaultGroupId int64 = 2
)

// 会员组对分类的权限 (参照苹果 CMS): 列表页、内容页、播放页、下载页、试看
const (
	PermList   = "list"
	PermDetail = "detail"
	PermPlay   = "play"
	PermDown   = "down"
	PermTrial  = "trial"
)

// AllPerms 全部分类权限
var AllPerms = []string{PermList, PermDetail, PermPlay, PermDown, PermTrial}

// Group 会员组 (member_groups 表)
type Group struct {
	Id         int64  `json:"id" gorm:"primaryKey"`
	Name       string `json:"name"`
	Status     bool   `json:"status"`
	PriceDay   int    `json:"priceDay"` // 包天 / 包周 / 包月 / 包年价格 (点数)
	PriceWeek  int    `json:"priceWeek"`
	PriceMonth int    `json:"priceMonth"`
	PriceYear  int    `json:"priceYear"`
	Remark     string `json:"remark"`
	// Permissions 分类ID → 权限列表; 没有设定的分类视为全部允许
	Permissions map[int64][]string `json:"permissions" gorm:"serializer:json"`
	CreatedAt   time.Time          `json:"-"`
	UpdatedAt   time.Time          `json:"-"`
}

// TableName 会员组表表名
func (Group) TableName() string {
	return "member_groups"
}

// Builtin 是否为内置会员组 (不能删除或停用)
func (g Group) Builtin() bool {
	return g.Id == GuestGroupId || g.Id == DefaultGroupId
}

// Allows 会员组对分类是否有某个权限 (没有设定的分类视为允许)
func (g Group) Allows(categoryId int64, perm string) bool {
	perms, ok := g.Permissions[categoryId]
	if !ok {
		return true
	}
	for _, p := range perms {
		if p == perm {
			return true
		}
	}
	return false
}

// Member 会员 (members 表)
type Member struct {
	Id          int64      `json:"id" gorm:"primaryKey"`
	UserName    string     `json:"userName"`
	Password    string     `json:"-"`
	Salt        string     `json:"-"`
	NickName    string     `json:"nickName"`
	Email       string     `json:"email"`
	GroupId     int64      `json:"groupId"`
	Points      int        `json:"points"`
	ExpireAt    *time.Time `json:"expireAt"` // 会员组到期时间, 为空表示不过期
	Status      bool       `json:"status"`
	LastLoginAt *time.Time `json:"lastLoginAt"`
	LastLoginIp string     `json:"lastLoginIp"`
	LoginCount  int        `json:"loginCount"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"-"`
}

// TableName 会员表表名
func (Member) TableName() string {
	return "members"
}
