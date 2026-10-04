package i18n

import (
	"errors"
	"log"
	"slices"
	"sync"
)

var Svc *Service

// Service 语言服务; 语言列表每个前台请求都要用, 缓存在内存中, 修改后重新加载
type Service struct {
	repo   *Repository
	mu     sync.RWMutex
	cache  []Language
	loaded bool
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// List 全部语言 (排序值小的在前)
func (s *Service) List() []Language {
	s.mu.RLock()
	if s.loaded {
		defer s.mu.RUnlock()
		return slices.Clone(s.cache)
	}
	s.mu.RUnlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.loaded {
		l, err := s.repo.List()
		if err != nil {
			log.Println("List Languages Error:", err)
			return nil
		}
		s.cache, s.loaded = l, true
	}
	return slices.Clone(s.cache)
}

// invalidate 修改语言后清除缓存
func (s *Service) invalidate() {
	s.mu.Lock()
	s.loaded = false
	s.mu.Unlock()
}

// Enabled 语言是否存在且已启用 (原文语言总是启用)
func (s *Service) Enabled(code string) bool {
	if code == SourceLang {
		return true
	}
	for _, l := range s.List() {
		if l.Code == code {
			return l.Enabled
		}
	}
	return false
}

// Save 新增 (isNew) 或修改语言; 原文语言不能停用
func (s *Service) Save(l Language, isNew bool) error {
	if err := ValidateLanguage(&l); err != nil {
		return err
	}
	if l.Code == SourceLang {
		l.Enabled = true
	}
	var err error
	if isNew {
		if s.repo.Exists(l.Code) {
			return errors.New("语言代码已存在")
		}
		err = s.repo.Create(&l)
	} else {
		err = s.repo.Update(&l)
	}
	s.invalidate()
	return err
}

// SetEnabled 启用 / 停用语言; 原文语言不能停用
func (s *Service) SetEnabled(code string, enabled bool) error {
	if code == SourceLang && !enabled {
		return errors.New("原文语言不能停用")
	}
	err := s.repo.SetEnabled(code, enabled)
	s.invalidate()
	return err
}

// ForScheme 请求使用的语言与前台可切换的语言 (见 Resolve / Available)
func (s *Service) ForScheme(schemeDefault string, schemeLangs []string, cookie string) (string, []Language) {
	return Resolve(cookie, schemeDefault, schemeLangs, s.Enabled), Available(s.List(), schemeLangs)
}

// CheckSchemeLangs 校验分类方案的语言设置 (见 CheckSchemeLangs)
func (s *Service) CheckSchemeLangs(def string, langs []string) (string, []string, error) {
	return CheckSchemeLangs(def, langs, s.Enabled)
}
