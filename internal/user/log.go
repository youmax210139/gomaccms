package user

import (
	"gomaccms/internal/db"
	"gomaccms/internal/paging"
	"log"
	"time"
)

// AdminLog 管理员操作日志 (admin_logs 表)
type AdminLog struct {
	Id        uint64    `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"createdAt"`
	UserId    uint      `json:"userId"`
	UserName  string    `json:"userName"`
	Ip        string    `json:"ip"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	Action    string    `json:"action"`
	Params    string    `json:"params"`
	Success   bool      `json:"success"`
}

// TableName 操作日志表表名
func (AdminLog) TableName() string {
	return "admin_logs"
}

// 登录日志的操作名称
const (
	ActionLogin       = "登录"
	ActionLoginFailed = "登录失败"
	ActionLogout      = "退出登录"
)

// LogQuery 操作日志筛选条件
type LogQuery struct {
	UserName string       `json:"userName"`
	Keyword  string       `json:"keyword"` // 操作名称 / 路径 / IP
	Success  string       `json:"success"` // "" 全部 / 1 成功 / 0 失败
	Paging   *paging.Page `json:"paging"`
}

// AddLog 写入一条操作日志 (失败只记录到程序日志, 不影响操作本身)
func (s *Service) AddLog(l *AdminLog) {
	if err := db.Mdb.Create(l).Error; err != nil {
		log.Println("Add admin log failed:", err)
	}
}

// ListLogs 按条件分页查询操作日志 (新的在前)
func (s *Service) ListLogs(q LogQuery) []AdminLog {
	query := db.Mdb.Model(&AdminLog{})
	if q.UserName != "" {
		query = query.Where("user_name = ?", q.UserName)
	}
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		query = query.Where("action LIKE ? OR path LIKE ? OR ip LIKE ?", like, like, like)
	}
	switch q.Success {
	case "1":
		query = query.Where("success = ?", true)
	case "0":
		query = query.Where("success = ?", false)
	}
	paging.Apply(query, q.Paging)
	var l []AdminLog
	if err := paging.Limit(query.Order("id DESC"), q.Paging).Find(&l).Error; err != nil {
		log.Println("List admin logs failed:", err)
	}
	return l
}

// LogUserNames 日志中出现过的账号 (筛选用)
func (s *Service) LogUserNames() []string {
	var names []string
	db.Mdb.Model(&AdminLog{}).Distinct("user_name").Order("user_name").Pluck("user_name", &names)
	return names
}

// ClearLogs 删除 days 天前的日志 (days 为 0 时删除全部), 返回删除的条数
func (s *Service) ClearLogs(days int) (int64, error) {
	query := db.Mdb.Where("1 = 1")
	if days > 0 {
		query = db.Mdb.Where("created_at < ?", time.Now().AddDate(0, 0, -days))
	}
	res := query.Delete(&AdminLog{})
	return res.RowsAffected, res.Error
}

// LogLogin 记录一次登录 (err 为登录失败的原因)
func (s *Service) LogLogin(account, ip string, err error) {
	l := &AdminLog{UserName: account, Ip: ip, Method: "POST", Path: "/login", Action: ActionLogin, Success: err == nil}
	if u := s.userRepo.FindByNameOrEmail(account); u != nil {
		l.UserId, l.UserName = u.ID, u.UserName
	}
	if err != nil {
		l.Action, l.Params = ActionLoginFailed, err.Error()
	}
	s.AddLog(l)
}
