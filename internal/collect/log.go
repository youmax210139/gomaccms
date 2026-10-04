package collect

import (
	"gomaccms/internal/db"
	"log"
	"time"
)

// CollectLog 采集历史 (collect_logs 表): 每次采集结束记录一笔
type CollectLog struct {
	Id          uint64       `json:"id" gorm:"primaryKey"`
	SourceId    string       `json:"sourceId"`
	SourceName  string       `json:"sourceName"`
	Mode        string       `json:"mode"`
	Trigger     string       `json:"trigger" gorm:"column:trigger"`
	Hours       int          `json:"hours"`
	State       string       `json:"state"`
	Added       int          `json:"added"`
	Updated     int          `json:"updated"`
	Skipped     int          `json:"skipped"`
	FailedPages int          `json:"failedPages"`
	FailedList  []FailedPage `json:"failedList" gorm:"serializer:json"`
	Retried     bool         `json:"retried"`
	Message     string       `json:"message"`
	StartedAt   time.Time    `json:"startedAt"`
	FinishedAt  time.Time    `json:"finishedAt"`
}

// FailedPage 采集失败的一页 (采集站分类 + 页码), 可在采集记录中重试
type FailedPage struct {
	TypeId int64 `json:"typeId"`
	Page   int   `json:"page"`
}

// TableName 采集历史表表名
func (CollectLog) TableName() string {
	return "collect_logs"
}

// keepCollectLogs 每个采集接口保留的采集历史笔数
const keepCollectLogs = 50

// SaveCollectLog 记录一次采集, 并删除该采集接口较早的记录
func (r *Repository) SaveCollectLog(l CollectLog) {
	if err := db.Mdb.Create(&l).Error; err != nil {
		log.Println("Save collect log failed:", err)
		return
	}
	var ids []uint64
	db.Mdb.Model(&CollectLog{}).Where("source_id = ?", l.SourceId).Order("id DESC").Offset(keepCollectLogs).Pluck("id", &ids)
	if len(ids) > 0 {
		db.Mdb.Where("id IN ?", ids).Delete(&CollectLog{})
	}
}

// CollectLogs 采集历史 (新的在前); sourceId 为空时为全部采集接口
func (r *Repository) CollectLogs(sourceId string, limit int) []CollectLog {
	var l []CollectLog
	query := db.Mdb.Order("id DESC").Limit(limit)
	if sourceId != "" {
		query = query.Where("source_id = ?", sourceId)
	}
	if err := query.Find(&l).Error; err != nil {
		log.Println("List collect logs failed:", err)
	}
	return l
}

// FindCollectLog 按 ID 查询一笔采集历史
func (r *Repository) FindCollectLog(id uint64) (CollectLog, error) {
	var l CollectLog
	err := db.Mdb.First(&l, id).Error
	return l, err
}

// MarkCollectLogRetried 标记采集历史的失败页已重试
func (r *Repository) MarkCollectLogRetried(id uint64) {
	if err := db.Mdb.Model(&CollectLog{}).Where("id = ?", id).Update("retried", true).Error; err != nil {
		log.Println("Mark collect log retried failed:", err)
	}
}
