package member

import (
	"gomaccms/internal/db"
	"gomaccms/internal/paging"
	"log"

	"gorm.io/gorm"
)

var Repo *Repository

type Repository struct{}

func NewRepository() *Repository {
	return &Repository{}
}

// ListGroups 全部会员组 (内置在前)
func (r *Repository) ListGroups() []Group {
	var l []Group
	if err := db.Mdb.Order("id").Find(&l).Error; err != nil {
		log.Println("List member groups failed:", err)
	}
	return l
}

// FindGroup 按ID查找会员组
func (r *Repository) FindGroup(id int64) (Group, bool) {
	var g Group
	res := db.Mdb.Where("id = ?", id).Limit(1).Find(&g)
	return g, res.Error == nil && res.RowsAffected > 0
}

func (r *Repository) CreateGroup(g *Group) error {
	return db.Mdb.Create(g).Error
}

// UpdateGroup 更新会员组 (columns 指定要更新的栏位)
func (r *Repository) UpdateGroup(g *Group, columns ...string) error {
	return db.Mdb.Model(&Group{}).Where("id = ?", g.Id).Select(columns).Updates(g).Error
}

// DeleteGroups 删除会员组, 其会员改为默认会员
func (r *Repository) DeleteGroups(ids []int64) error {
	return db.Mdb.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&Member{}).Where("group_id IN ?", ids).Update("group_id", DefaultGroupId).Error; err != nil {
			return err
		}
		return tx.Where("id IN ?", ids).Delete(&Group{}).Error
	})
}

// MemberQuery 会员列表筛选条件
type MemberQuery struct {
	GroupId int64        `json:"groupId"`
	Status  string       `json:"status"` // "" 全部 / 1 启用 / 0 停用
	Keyword string       `json:"keyword"`
	Paging  *paging.Page `json:"paging"`
}

// ListMembers 按条件分页查询会员 (新的在前)
func (r *Repository) ListMembers(q MemberQuery) []Member {
	query := db.Mdb.Model(&Member{})
	if q.GroupId > 0 {
		query = query.Where("group_id = ?", q.GroupId)
	}
	switch q.Status {
	case "1":
		query = query.Where("status = ?", true)
	case "0":
		query = query.Where("status = ?", false)
	}
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		query = query.Where("user_name LIKE ? OR nick_name LIKE ? OR email LIKE ?", like, like, like)
	}
	paging.Apply(query, q.Paging)
	var l []Member
	if err := paging.Limit(query.Order("id DESC"), q.Paging).Find(&l).Error; err != nil {
		log.Println("List members failed:", err)
	}
	return l
}

// FindMember 按ID查找会员
func (r *Repository) FindMember(id int64) (Member, bool) {
	var m Member
	res := db.Mdb.Where("id = ?", id).Limit(1).Find(&m)
	return m, res.Error == nil && res.RowsAffected > 0
}

// UserNameTaken 会员账号是否已被 exceptId 以外的会员使用
func (r *Repository) UserNameTaken(name string, exceptId int64) bool {
	var count int64
	db.Mdb.Model(&Member{}).Where("user_name = ? AND id <> ?", name, exceptId).Count(&count)
	return count > 0
}

func (r *Repository) CreateMember(m *Member) error {
	return db.Mdb.Create(m).Error
}

// UpdateMember 更新会员 (columns 指定要更新的栏位)
func (r *Repository) UpdateMember(m *Member, columns ...string) error {
	return db.Mdb.Model(&Member{}).Where("id = ?", m.Id).Select(columns).Updates(m).Error
}

func (r *Repository) DeleteMembers(ids []int64) error {
	return db.Mdb.Where("id IN ?", ids).Delete(&Member{}).Error
}

// GroupMemberCounts 各会员组的会员数
func (r *Repository) GroupMemberCounts() map[int64]int64 {
	var rows []struct {
		GroupId int64
		Total   int64
	}
	db.Mdb.Model(&Member{}).Select("group_id, COUNT(*) AS total").Group("group_id").Scan(&rows)
	m := make(map[int64]int64, len(rows))
	for _, row := range rows {
		m[row.GroupId] = row.Total
	}
	return m
}
