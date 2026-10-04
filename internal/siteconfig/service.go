package siteconfig

var Svc *Service

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetSiteBasicConfig() BasicConfig {
	return s.repo.GetSiteBasic()
}

func (s *Service) UpdateSiteBasic(c BasicConfig) error {
	return s.repo.SaveSiteBasic(c)
}

// AdminPageSize 后台列表每页条数
func (s *Service) AdminPageSize() int {
	return s.repo.GetSiteBasic().AdminPageSize()
}

// HasSiteBasic Redis 中是否已有站点配置 (没有时由 bootstrap 写入默认值)
func (s *Service) HasSiteBasic() bool {
	return s.repo.HasSiteBasic()
}
