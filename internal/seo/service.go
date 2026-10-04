package seo

import (
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"gomaccms/internal/db"

	"gorm.io/gorm"
)

// Rule SEO 规则 (seo_rules 表): 分类方案 + 页面类型 + 语言 唯一
type Rule struct {
	Id          int64  `gorm:"primaryKey"`
	SchemeId    int64  `gorm:"column:scheme_id"`
	PageType    string `gorm:"column:page_type"`
	Lang        string `gorm:"column:lang"`
	Title       string
	Keywords    string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// TableName SEO 规则表表名
func (Rule) TableName() string {
	return "seo_rules"
}

// Rules 一个分类方案的规则: 页面类型 → 语言 → 文字 (后台编辑与保存的格式)
type Rules map[string]map[string]Text

// 栏位长度上限 (与表结构一致)
const (
	maxTitle       = 255
	maxKeywords    = 255
	maxDescription = 500
)

var Svc = &Service{cache: map[int64]Rules{}}

// Service 读取 (按方案缓存在内存中) 与保存 SEO 规则
type Service struct {
	mu    sync.RWMutex
	cache map[int64]Rules
}

// Rules 分类方案的全部规则 (后台)
func (s *Service) Rules(schemeId int64) Rules {
	s.mu.RLock()
	r, ok := s.cache[schemeId]
	s.mu.RUnlock()
	if ok {
		return r
	}
	var list []Rule
	if err := db.Mdb.Where("scheme_id = ?", schemeId).Find(&list).Error; err != nil {
		return Rules{} // 读取失败时不缓存, 页面使用站点 SEO
	}
	r = Rules{}
	for _, row := range list {
		if r[row.PageType] == nil {
			r[row.PageType] = map[string]Text{}
		}
		r[row.PageType][row.Lang] = Text{Title: row.Title, Keywords: row.Keywords, Description: row.Description}
	}
	s.mu.Lock()
	s.cache[schemeId] = r
	s.mu.Unlock()
	return r
}

// Rule 分类方案中某页面类型、某语言的规则 (没有时为空)
func (s *Service) Rule(schemeId int64, pageType, lang string) Text {
	return s.Rules(schemeId)[pageType][lang]
}

// Save 以 rules 替换分类方案的全部规则 (三个栏位都空的不保存), 并清除缓存
func (s *Service) Save(schemeId int64, rules Rules) error {
	list, err := rows(schemeId, rules)
	if err != nil {
		return err
	}
	err = db.Mdb.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("scheme_id = ?", schemeId).Delete(&Rule{}).Error; err != nil {
			return err
		}
		if len(list) == 0 {
			return nil
		}
		return tx.Create(&list).Error
	})
	s.Clear()
	return err
}

// Clear 清除规则缓存 (保存规则或删除方案后)
func (s *Service) Clear() {
	s.mu.Lock()
	s.cache = map[int64]Rules{}
	s.mu.Unlock()
}

// rows 校验并转换要保存的规则: 只接受已知的页面类型, 去掉首尾空白, 跳过全空的
func rows(schemeId int64, rules Rules) ([]Rule, error) {
	var list []Rule
	for pageType, langs := range rules {
		if !slices.Contains(PageTypes, pageType) {
			continue
		}
		for lang, t := range langs {
			t = Text{Title: strings.TrimSpace(t.Title), Keywords: strings.TrimSpace(t.Keywords), Description: strings.TrimSpace(t.Description)}
			if t == (Text{}) || lang == "" || len(lang) > 16 {
				continue
			}
			switch {
			case utf8.RuneCountInString(t.Title) > maxTitle:
				return nil, fmt.Errorf("%s (%s) 的 Title 不能超过 %d 个字", pageType, lang, maxTitle)
			case utf8.RuneCountInString(t.Keywords) > maxKeywords:
				return nil, fmt.Errorf("%s (%s) 的 Keywords 不能超过 %d 个字", pageType, lang, maxKeywords)
			case utf8.RuneCountInString(t.Description) > maxDescription:
				return nil, fmt.Errorf("%s (%s) 的 Description 不能超过 %d 个字", pageType, lang, maxDescription)
			}
			list = append(list, Rule{SchemeId: schemeId, PageType: pageType, Lang: lang, Title: t.Title, Keywords: t.Keywords, Description: t.Description})
		}
	}
	return list, nil
}
