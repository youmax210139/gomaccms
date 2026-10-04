package collect

import (
	"errors"
	"gomaccms/internal/util"
	"net/url"
	"regexp"
	"strings"
)

var Svc *Service

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetFilmSourceList() []FilmSource {
	return s.repo.GetCollectSourceList()
}

func (s *Service) GetFilmSource(id string) *FilmSource {
	return s.repo.FindCollectSourceById(id)
}

func (s *Service) UpdateFilmSource(fs FilmSource) error {
	return s.repo.UpdateCollectSource(fs)
}

// ChangeFilmSourceState 启用/停用采集接口
func (s *Service) ChangeFilmSourceState(id string, state bool) error {
	if s.repo.FindCollectSourceById(id) == nil {
		return errors.New("采集接口信息不存在")
	}
	return s.repo.UpdateCollectSourceState(id, state)
}

func (s *Service) SaveFilmSource(fs FilmSource) error {
	return s.repo.AddCollectSource(fs)
}

func (s *Service) DelFilmSource(id string) error {
	fs := s.repo.FindCollectSourceById(id)
	if fs == nil {
		return errors.New("当前资源站信息不存在, 请勿重复操作")
	}
	s.repo.DelCollectResource(id)
	_ = s.repo.ClearBinds(id)
	return nil
}

// GetBindMap 采集站分类绑定 TypeId → 本站分类ID 列表
func (s *Service) GetBindMap(sourceId string) map[int64][]int64 {
	return s.repo.GetBindMap(sourceId)
}

// Bind 覆盖采集站分类绑定的本站分类, categoryIds 为空时解除绑定
func (s *Service) Bind(sourceId string, typeId int64, typeName string, categoryIds []int64) error {
	return s.repo.SetBinds(sourceId, typeId, typeName, categoryIds)
}

// ClearBinds 清空指定采集站的分类绑定
func (s *Service) ClearBinds(sourceIds ...string) error {
	return s.repo.ClearBinds(sourceIds...)
}

// NormalizeFilmSource 去除表单输入中多余的空白
func NormalizeFilmSource(fs *FilmSource) {
	fs.Name = strings.TrimSpace(fs.Name)
	fs.Uri = strings.TrimSpace(fs.Uri)
	fs.Params = strings.TrimSpace(fs.Params)
	fs.FilterCode = strings.ReplaceAll(strings.TrimSpace(fs.FilterCode), " ", "")
	fs.FilterYear = strings.ReplaceAll(strings.TrimSpace(fs.FilterYear), " ", "")
}

var filterYearReg = regexp.MustCompile(`^\d{4}(,\d{4})*$`)

func (s *Service) ValidFilmSource(fs FilmSource) error {
	if len(fs.Name) <= 0 || len(fs.Name) > 20 {
		return errors.New("资源名称不能为空且长度不能超过20")
	}
	if !util.ValidURL(fs.Uri) {
		return errors.New("资源链接格式异常, 请输入规范的URL链接")
	}
	if _, err := url.ParseQuery(strings.TrimLeft(fs.Params, "&?")); err != nil || len(fs.Params) > 500 {
		return errors.New("附加参数格式异常, 例如: &ct=1")
	}
	if fs.ResultModel != JsonResult && fs.ResultModel != XmlResult {
		return errors.New("接口类型异常, 请提交正确的接口类型")
	}
	switch fs.CollectType {
	case CollectVideo, CollectArticle, CollectActor, CollectRole, CollectWebSite:
	default:
		return errors.New("资源类型异常, 未知的资源类型")
	}
	if fs.Operation < OperationAll || fs.Operation > OperationUpdate {
		return errors.New("数据操作异常")
	}
	if fs.FilterMode < FilterNone || fs.FilterMode > FilterUpdate {
		return errors.New("地址过滤异常")
	}
	if len(fs.FilterCode) > 255 {
		return errors.New("过滤代码长度不能超过255")
	}
	if fs.FilterYear != "" && (!filterYearReg.MatchString(fs.FilterYear) || len(fs.FilterYear) > 255) {
		return errors.New("过滤年份格式异常, 多个年份用英文逗号分隔, 如 2022,2023")
	}
	if fs.SyncImage < SyncImageGlobal || fs.SyncImage > SyncImageOff {
		return errors.New("同步图片设置异常")
	}
	return nil
}
