package film

import (
	"errors"
	"fmt"
	"gomaccms/internal/i18n"
	"regexp"
	"strings"
	"unicode/utf8"
)

var CategorySvc *CategoryService

type CategoryService struct {
	repo *CategoryRepository
}

func NewCategoryService(repo *CategoryRepository) *CategoryService {
	return &CategoryService{repo: repo}
}

// slugPattern slug 只允许小写字母、数字与中划线, 如 sci-fi
var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// GetFilmClassTree 获取视频分类树
func (s *CategoryService) GetFilmClassTree() CategoryTree {
	return s.repo.GetCategoryTree()
}

// GetAllClassTree 分类方案中全部类型的分类树 (后台分类管理)
func (s *CategoryService) GetAllClassTree(schemeId int64) CategoryTree {
	return BuildTree(s.repo.List(0, schemeId))
}

// GetFilmClassById 通过ID获取视频分类信息 (在其所属方案的分类树中查找)
func (s *CategoryService) GetFilmClassById(id int64) *CategoryTree {
	c, err := s.repo.Find(id)
	if err != nil {
		return nil
	}
	tree := s.repo.GetSchemeCategoryTree(c.SchemeId)
	node, _ := findClass(&tree, id)
	return node
}

// findClass 在分类树中查找指定ID的分类及其父节点 (一级分类的父节点为树根)
func findClass(tree *CategoryTree, id int64) (node, parent *CategoryTree) {
	for _, c := range tree.Children {
		if c.Id == id {
			return c, tree
		}
		for _, subC := range c.Children {
			if subC.Id == id {
				return subC, c
			}
		}
	}
	return nil, nil
}

// SaveClass 新增 (Id 为 0) 或更新分类. 分类最多两级: 上级分类必须是同类型的一级分类
func (s *CategoryService) SaveClass(c *Category) error {
	c.Name = strings.TrimSpace(c.Name)
	c.Slug = strings.TrimSpace(c.Slug)
	switch {
	case CategoryTypes[c.Type] == "":
		return errors.New("分类类型无效")
	case c.Sort < 0:
		return ErrNegativeSort
	case c.Name == "" || utf8.RuneCountInString(c.Name) > 60:
		return errors.New("分类名称不能为空, 且不能超过 60 个字符")
	case len(c.Slug) > 60 || !slugPattern.MatchString(c.Slug):
		return errors.New("Slug 不能为空, 只能包含小写字母、数字与中划线 (如 sci-fi), 且不能超过 60 个字符")
	}
	// 所属方案在创建时决定, 之后不变
	var old *Category
	if c.Id != 0 {
		var err error
		if old, err = s.repo.Find(c.Id); err != nil {
			return errors.New("需要更新的分类信息不存在")
		}
		c.SchemeId = old.SchemeId
		c.I18n = mergeCategoryI18n(old.I18n, c.I18n)
	} else {
		c.I18n = normalizeCategoryI18n(c.I18n)
		if c.SchemeId == 0 {
			c.SchemeId = DefaultSchemeId
		}
	}
	if !s.repo.SchemeExists(c.SchemeId) {
		return errors.New("分类方案不存在")
	}
	if s.repo.SlugTaken(c.Slug, c.Id, c.SchemeId) {
		return fmt.Errorf("Slug %q 已被本方案中的其他分类使用", c.Slug)
	}
	if c.Pid != 0 {
		parent, err := s.repo.Find(c.Pid)
		switch {
		case err != nil || parent.Id == c.Id:
			return errors.New("上级分类不存在")
		case parent.SchemeId != c.SchemeId:
			return errors.New("上级分类必须属于同一个分类方案")
		case parent.Type != c.Type:
			return errors.New("只能选择相同类型的分类作为上级分类")
		case parent.Pid != 0:
			return errors.New("上级分类必须是一级分类")
		}
	}
	if old == nil {
		return s.repo.Create(c)
	}
	if s.repo.HasChildren(c.Id) && (c.Pid != 0 || c.Type != old.Type) {
		return errors.New("该分类下有子分类, 不能修改其类型或设为二级分类")
	}
	if err := s.syncFilmShield(old, c.Show); err != nil {
		return err
	}
	return s.repo.Update(c)
}

// SetClassShow 修改分类的启用状态
func (s *CategoryService) SetClassShow(id int64, show bool) error {
	defer VodsChanged(nil, nil)
	c, err := s.repo.Find(id)
	if err != nil {
		return errors.New("需要更新的分类信息不存在")
	}
	if err := s.syncFilmShield(c, show); err != nil {
		return err
	}
	return s.repo.UpdateColumn(id, "status", show)
}

// SetClassSort 修改分类排序
func (s *CategoryService) SetClassSort(id int64, sort int64) error {
	if sort < 0 {
		return ErrNegativeSort
	}
	return s.repo.UpdateColumn(id, "sort", sort)
}

// syncFilmShield 视频二级分类的启用状态变化时, 同步屏蔽/恢复该分类下的影片检索信息
func (s *CategoryService) syncFilmShield(c *Category, show bool) error {
	if c.Type != CategoryVideo || c.Pid == 0 || c.Show == show {
		return nil
	}
	if show {
		return Svc.RecoverFilmSearch(c.Id)
	}
	return Svc.ShieldFilmSearch(c.Id)
}

// DelClass 删除分类及其子分类 (分类下的影片不会删除)
func (s *CategoryService) DelClass(id int64) error {
	defer VodsChanged(nil, nil)
	if _, err := s.repo.Find(id); err != nil {
		return errors.New("需要删除的分类信息不存在")
	}
	if err := s.repo.Delete(id); err != nil {
		return fmt.Errorf("分类删除失败: %s", err.Error())
	}
	return nil
}

// FilmCounts 统计分类方案中每个视频分类下的影片数 (含停用分类中的影片): 二级分类按 cid, 一级分类按 pid
func (s *CategoryService) FilmCounts(schemeId int64) map[int64]int64 {
	byCid, byPid := SearchRepo.CountByCategory()
	tree := s.repo.GetSchemeCategoryTree(schemeId)
	counts := make(map[int64]int64)
	for _, c := range tree.Children {
		counts[c.Id] = byPid[c.Id]
		for _, subC := range c.Children {
			counts[subC.Id] = byCid[subC.Id]
		}
	}
	return counts
}

// TransferFilms 将 ids 分类下的影片转移到 target 视频二级分类 (一级分类转移其下全部影片);
// 只在 target 所属的分类方案内转移
func (s *CategoryService) TransferFilms(ids []int64, target int64) error {
	t, err := s.repo.Find(target)
	if err != nil {
		return errors.New("目标分类不存在或不是视频二级分类")
	}
	tree := s.repo.GetSchemeCategoryTree(t.SchemeId)
	to, _ := findClass(&tree, target)
	if to == nil || to.Pid == 0 {
		return errors.New("目标分类不存在或不是视频二级分类")
	}
	for _, id := range ids {
		from, _ := findClass(&tree, id)
		if from == nil || from.Id == target {
			continue
		}
		column := "cid"
		if from.Pid == 0 {
			column = "pid"
		}
		if err := SearchRepo.TransferCategory(column, from.Id, *to.Category); err != nil {
			return fmt.Errorf("视频转移失败: %s", err.Error())
		}
	}
	return nil
}

// ------------------------------------------------ 分类方案 ------------------------------------------------

// ListSchemes 全部分类方案 (默认方案在前)
func (s *CategoryService) ListSchemes() []CategoryScheme {
	return s.repo.ListSchemes()
}

// SchemeExists 分类方案是否存在
func (s *CategoryService) SchemeExists(id int64) bool {
	return s.repo.SchemeExists(id)
}

// SaveScheme 新增 (Id 为 0) 或修改分类方案 (名称、排序、语言设置)
func (s *CategoryService) SaveScheme(sc *CategoryScheme) error {
	sc.Name = strings.TrimSpace(sc.Name)
	if sc.Name == "" || utf8.RuneCountInString(sc.Name) > 60 {
		return errors.New("方案名称不能为空, 且不能超过 60 个字符")
	}
	def, langs, err := i18n.Svc.CheckSchemeLangs(sc.DefaultLang, sc.Langs)
	if err != nil {
		return err
	}
	sc.DefaultLang, sc.Langs = def, langs
	if sc.Sort < 0 {
		return ErrNegativeSort
	}
	return s.repo.SaveScheme(sc)
}

// Scheme 分类方案 (不存在时为零值, 由调用方回退到原文语言)
func (s *CategoryService) Scheme(id int64) CategoryScheme {
	sc, err := s.repo.FindScheme(id)
	if err != nil {
		return CategoryScheme{}
	}
	return sc
}

// DeleteScheme 删除分类方案及其全部分类 (默认方案不能删除; 是否有域名在使用由调用方检查)
func (s *CategoryService) DeleteScheme(id int64) error {
	defer VodsChanged(nil, nil)
	if id == DefaultSchemeId {
		return errors.New("默认方案不能删除")
	}
	if !s.repo.SchemeExists(id) {
		return errors.New("分类方案不存在")
	}
	return s.repo.DeleteScheme(id)
}

// SchemeCategoryCounts 各分类方案的分类数 (方案管理页)
func (s *CategoryService) SchemeCategoryCounts() map[int64]int64 {
	return s.repo.CountBySchemes()
}
