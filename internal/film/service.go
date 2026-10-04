package film

import (
	"errors"
	"time"
)

var Svc *Service

type Service struct {
	movieRepo  *MovieRepository
	searchRepo *SearchRepository
}

func NewService(movieRepo *MovieRepository, searchRepo *SearchRepository) *Service {
	return &Service{movieRepo: movieRepo, searchRepo: searchRepo}
}

// GetFilmPage 获取影片检索信息分页数据
func (s *Service) GetFilmPage(sv SearchVo) []SearchInfo {
	return s.searchRepo.GetSearchPage(sv)
}

// GetSearchOptions 获取影片检索的select的选项options (分类为 schemeId 方案中的分类)
func (s *Service) GetSearchOptions(schemeId int64) map[string]any {
	var options = make(map[string]any)
	options["schemes"] = CategorySvc.ListSchemes()
	options["players"] = ListPlayers()
	tree := CategoryRepo.GetSchemeCategoryTree(schemeId)
	// 尚未采集过分类时 redis 中没有分类树, 反序列化得到的 tree.Category 为 nil
	if tree.Category == nil {
		tree.Category = &Category{}
	}
	tree.Name = "全部分类"
	options["class"] = ConvertCategoryList(tree)
	options["remarks"] = []map[string]string{{"Name": `全部`, "Value": ``}, {"Name": `完结`, "Value": `完结`}, {"Name": `未完结`, "Value": `未完结`}}
	var tagGroup = make(map[int64]map[string]any)
	for _, t := range tree.Children {
		option := s.searchRepo.GetSearchOptions(t.Id)
		if len(option) > 0 {
			tagGroup[t.Id] = s.searchRepo.GetSearchOptions(t.Id)
			if _, ok := options["year"]; !ok {
				options["year"] = tagGroup[t.Id]["Year"]
			}
		}
	}
	options["tags"] = tagGroup
	return options
}

// SaveFilmDetail 自定义上传保存影片信息
func (s *Service) SaveFilmDetail(fd FilmDetailVo) error {
	now := time.Now()
	fd.UpdateTime = now.Format(time.DateTime)
	fd.AddTime = fd.UpdateTime
	fd.Id = 0 // 由数据库分配影片ID
	detail, err := CovertFilmDetailVo(fd)
	if err != nil || detail.PlayList == nil {
		return errors.New("视频参数格式异常或缺少关键信息")
	}
	return s.movieRepo.SaveDetail(detail)
}

// DelFilm 删除分类影片
func (s *Service) DelFilm(id int64) error {
	si := s.searchRepo.GetSearchInfoById(id)
	if si == nil {
		return errors.New("视频信息不存在")
	}
	return s.searchRepo.DelFilmSearch(id)
}

// ShieldFilmSearch 删除所属分类下的所有影片检索信息(供 CategoryService.UpdateClass 调用)
func (s *Service) ShieldFilmSearch(cid int64) error {
	return s.searchRepo.ShieldFilmSearch(cid)
}

// RecoverFilmSearch 恢复所属分类下的影片检索信息状态(供 CategoryService.UpdateClass 调用)
func (s *Service) RecoverFilmSearch(cid int64) error {
	return s.searchRepo.RecoverFilmSearch(cid)
}

// GetNamesByMids 按影片 mid 批量获取影片名称
func (s *Service) GetNamesByMids(mids []int64) map[int64]string {
	return s.searchRepo.GetNamesByMids(mids)
}

// GetMovieListByPid/GetMovieListByCid/GetRelateMovieBasicInfo/GetMultiplePlay/
// GetMovieListBySort/SearchFilmKeyword/GetSearchInfosByTags/GetSearchTag
// 这些方法由 IndexService 直接呼叫
// MovieRepo/SearchRepo,不透过 FilmService 转发——
// 首页/搜索的资料读取只属于 Index 网域,不需要额外经过 FilmService 这层。
