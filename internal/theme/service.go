package theme

import (
	"encoding/json"
	"errors"
	"fmt"
	"gomaccms/internal/config"
	"gomaccms/internal/db"
	"gomaccms/internal/siteconfig"
	"log"
	"regexp"
	"strings"
)

var Svc *Service

const defaultThemeName = "default"

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// ThemeOf returns the theme a resolved domains record (possibly nil) renders
// with, falling back to "default".
func ThemeOf(d *Domain) string {
	if d != nil && d.Theme != "" {
		return d.Theme
	}
	return defaultThemeName
}

// ResolveDomain returns the domains record configured for a request Host
// (no port), or nil when the host isn't configured (callers then fall back to
// the default theme and the global site config).
func (s *Service) ResolveDomain(host string) *Domain {
	if d, ok := s.getDomainMap()[host]; ok {
		return &d
	}
	return nil
}

// getDomainMap 读取 Host → 域名记录 的缓存; 缓存缺失或格式不符 (如旧版本的 Host → 主题名) 时重建
func (s *Service) getDomainMap() map[string]Domain {
	data := db.Rdb.Get(db.Cxt, config.ThemeDomainMapKey).Val()
	if data != "" {
		var m map[string]Domain
		if err := json.Unmarshal([]byte(data), &m); err == nil {
			return m
		}
	}
	return s.rebuildDomainMapCache()
}

func (s *Service) rebuildDomainMapCache() map[string]Domain {
	domains, err := s.repo.GetAll()
	if err != nil {
		log.Println("ThemeService: failed to load domains:", err)
		return map[string]Domain{}
	}
	m := make(map[string]Domain, len(domains))
	for _, d := range domains {
		m[d.Domain] = d
	}
	data, _ := json.Marshal(m)
	if err := db.Rdb.Set(db.Cxt, config.ThemeDomainMapKey, data, config.ManageConfigExpired).Err(); err != nil {
		log.Println("ThemeService: failed to cache domain map:", err)
	}
	return m
}

// hostPattern 校验不含协议/端口/路径的 Host (与 ResolveTheme 比对的请求 Host 一致),
// 允许 localhost、IPv4 与多级子域名
var hostPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)*$`)

// NormalizeDomain 去除首尾空白并转为小写, 与请求 Host 的比对方式保持一致
func NormalizeDomain(domain string) string {
	return strings.ToLower(strings.TrimSpace(domain))
}

// AllDomains 获取全部域名映射 (后台设置页展示, 数量很少无需分页)
func (s *Service) AllDomains() []Domain {
	dl, err := s.repo.GetAll()
	if err != nil {
		log.Println("ThemeService: failed to list domains:", err)
	}
	return dl
}

// CreateDomain / UpdateDomain 的 d.Domain 需先经 NormalizeDomain 处理;
// d.Theme 是否为有效主题由调用方 (handler, 依据 renderer 的主题发现结果) 校验

func (s *Service) CreateDomain(d *Domain) error {
	if err := s.checkDomain(0, d.Domain); err != nil {
		return err
	}
	if err := s.repo.Create(d); err != nil {
		return err
	}
	s.rebuildDomainMapCache()
	return nil
}

func (s *Service) UpdateDomain(d *Domain) error {
	if err := s.checkDomain(d.ID, d.Domain); err != nil {
		return err
	}
	if err := s.repo.Update(d); err != nil {
		return err
	}
	s.rebuildDomainMapCache()
	return nil
}

// UpdateDomainText 更新域名的网站名称、SEO、法律信息及各语言译文
func (s *Service) UpdateDomainText(id uint, si siteconfig.SiteInfo) error {
	if err := s.repo.UpdateText(id, si); err != nil {
		return err
	}
	s.rebuildDomainMapCache()
	return nil
}

// SetDomainState 开启 / 关闭指定域名的网站
func (s *Service) SetDomainState(id uint, state bool) error {
	if err := s.repo.UpdateState(id, state); err != nil {
		return err
	}
	s.rebuildDomainMapCache()
	return nil
}

func (s *Service) DeleteDomain(id uint) error {
	if err := s.repo.Delete(id); err != nil {
		return err
	}
	s.rebuildDomainMapCache()
	return nil
}

// checkDomain 校验域名格式, 并确认没有被其他记录 (id 以外) 占用
func (s *Service) checkDomain(id uint, domain string) error {
	if domain == "" {
		return errors.New("域名不能为空")
	}
	if len(domain) > 253 || !hostPattern.MatchString(domain) {
		return errors.New("域名格式错误, 请填写不含 http:// 与端口的域名, 如 movie.example.com")
	}
	if d, err := s.repo.GetByDomain(domain); err == nil && d.ID != id {
		return fmt.Errorf("域名 %s 已存在", domain)
	}
	return nil
}
