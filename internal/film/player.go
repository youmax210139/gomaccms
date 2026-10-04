package film

import (
	"errors"
	"gomaccms/internal/db"
	"log"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm/clause"
)

// Player 播放器 (players 表, 参照苹果 CMS): 一个播放组代码对应的前台线路名称、顺序与是否显示
type Player struct {
	Code      string    `json:"code" gorm:"primaryKey"`
	Name      string    `json:"name"`
	Status    bool      `json:"status"` // 停用时前台不显示该线路
	Sort      int64     `json:"sort"`   // 越小越靠前
	Remark    string    `json:"remark"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

// TableName 播放器表表名
func (Player) TableName() string {
	return "players"
}

// ListPlayers 全部播放器 (按排序)
func ListPlayers() []Player {
	var l []Player
	if err := db.Mdb.Order("sort ASC, code ASC").Find(&l).Error; err != nil {
		log.Println("List Players Error:", err)
	}
	return l
}

// PlayerMap 播放组代码 → 播放器
func PlayerMap() map[string]Player {
	m := make(map[string]Player)
	for _, p := range ListPlayers() {
		m[p.Code] = p
	}
	return m
}

// EnsurePlayers 为尚未设置的播放组代码新增播放器 (名称为 name, 启用), 已有的不修改
func EnsurePlayers(codes []string, name string) {
	var l []Player
	for _, code := range codes {
		if code = strings.TrimSpace(code); code != "" {
			l = append(l, Player{Code: clip(code, 60), Name: clip(name, 60), Status: true})
		}
	}
	if len(l) == 0 {
		return
	}
	if err := db.Mdb.Clauses(clause.OnConflict{DoNothing: true}).Create(&l).Error; err != nil {
		log.Println("Ensure Players Error:", err)
	}
}

// SavePlayer 新增或修改播放器; isNew 为 true 时代码不能已存在
func SavePlayer(p *Player, isNew bool) error {
	p.Code, p.Name, p.Remark = strings.TrimSpace(p.Code), strings.TrimSpace(p.Name), strings.TrimSpace(p.Remark)
	switch {
	case p.Code == "" || utf8.RuneCountInString(p.Code) > 60:
		return errors.New("播放器代码不能为空, 且不能超过 60 个字符")
	case p.Name == "" || utf8.RuneCountInString(p.Name) > 60:
		return errors.New("播放器名称不能为空, 且不能超过 60 个字符")
	case utf8.RuneCountInString(p.Remark) > 255:
		return errors.New("备注不能超过 255 个字符")
	case p.Sort < 0:
		return ErrNegativeSort
	}
	var count int64
	db.Mdb.Model(&Player{}).Where("code = ?", p.Code).Count(&count)
	if isNew {
		if count > 0 {
			return errors.New("播放器代码已存在")
		}
		return db.Mdb.Create(p).Error
	}
	if count == 0 {
		return errors.New("播放器不存在")
	}
	return db.Mdb.Model(&Player{}).Where("code = ?", p.Code).Select("updated_at", "name", "status", "sort", "remark").Updates(p).Error
}

// SetPlayerStatus 启用 / 停用播放器
func SetPlayerStatus(code string, status bool) error {
	return db.Mdb.Model(&Player{}).Where("code = ?", code).Update("status", status).Error
}

// DeletePlayer 删除播放器 (影片的播放地址不删除, 前台线路名称改为显示代码)
func DeletePlayer(code string) error {
	return db.Mdb.Where("code = ?", code).Delete(&Player{}).Error
}

// PlayerFilmCounts 每个播放组代码被多少部影片使用
func PlayerFilmCounts() map[string]int64 {
	counts := make(map[string]int64)
	for _, p := range ListPlayers() {
		var n int64
		db.Mdb.Model(&Vod{}).Where("FIND_IN_SET(?, REPLACE(vod_play_from, '$$$', ','))", p.Code).Count(&n)
		counts[p.Code] = n
	}
	return counts
}
