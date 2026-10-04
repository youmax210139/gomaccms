package admin

import (
	"gomaccms/internal/collect"
	"gomaccms/internal/film"
	"gomaccms/internal/http/request"
	"gomaccms/internal/http/response"
	"gomaccms/internal/spider"
	"gomaccms/internal/view/inertia"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	gonertia "github.com/romsar/gonertia/v3"
)

// bindClass 采集站分类及其绑定的本站分类 (可多个)
type bindClass struct {
	TypeId      int64    `json:"typeId"`
	TypePid     int64    `json:"typePid"`
	TypeName    string   `json:"typeName"`
	CategoryIds []int64  `json:"categoryIds"`
	Categories  []string `json:"categories"` // 绑定的本站分类名称, 如「默认方案 / 动作片」
	// Inherited 自己没有绑定, 沿用一级分类的绑定 (Categories 为一级分类绑定的分类)
	Inherited bool `json:"inherited"`
}

// categoryOption 绑定弹窗中的本站分类选项 (按树形顺序展开, Level 从 0 开始)
type categoryOption struct {
	Id    int64  `json:"id"`
	Name  string `json:"name"`
	Level int    `json:"level"`
}

// categoryGroup 一个分类方案的分类选项
type categoryGroup struct {
	SchemeId int64            `json:"schemeId"`
	Name     string           `json:"name"`
	Options  []categoryOption `json:"options"`
}

// categoryOptions 按分类方案分组的视频分类选项, 以及 分类ID → 「方案 / 分类」显示名称
func categoryOptions() ([]categoryGroup, map[int64]string) {
	var groups []categoryGroup
	names := make(map[int64]string)
	for _, sc := range film.CategorySvc.ListSchemes() {
		g := categoryGroup{SchemeId: sc.Id, Name: sc.Name}
		tree := film.CategoryRepo.GetSchemeCategoryTree(sc.Id)
		var walk func(t *film.CategoryTree, level int)
		walk = func(t *film.CategoryTree, level int) {
			for _, c := range t.Children {
				g.Options = append(g.Options, categoryOption{Id: c.Id, Name: c.Name, Level: level})
				names[c.Id] = sc.Name + " / " + c.Name
				walk(c, level+1)
			}
		}
		walk(&tree, 0)
		groups = append(groups, g)
	}
	return groups, names
}

// CollectBrowse 浏览采集接口资源并绑定分类(Inertia 渲染)
func CollectBrowse(c *gin.Context) {
	s := collect.Svc.GetFilmSource(c.Query("id"))
	if s == nil {
		inertia.Redirect(c, "/manage/collect/list")
		return
	}
	t, _ := strconv.ParseInt(c.DefaultQuery("t", "0"), 10, 64)
	pg, _ := strconv.Atoi(c.DefaultQuery("pg", "1"))
	wd := strings.TrimSpace(c.Query("wd"))

	options, names := categoryOptions()
	binds := collect.Svc.GetBindMap(s.Id)
	page, err := spider.BrowseSource(*s, t, wd, pg)
	var classes []bindClass
	for _, fc := range page.Class {
		bc := bindClass{TypeId: int64(fc.TypeID), TypePid: int64(fc.TypePid), TypeName: fc.TypeName}
		// 绑定的本站分类已被删除时不显示
		for _, id := range binds[int64(fc.TypeID)] {
			if name, ok := names[id]; ok {
				bc.CategoryIds = append(bc.CategoryIds, id)
				bc.Categories = append(bc.Categories, name)
			}
		}
		// 没有单独绑定的子分类沿用一级分类的绑定 (采集时同样如此)
		if len(bc.CategoryIds) == 0 && fc.TypePid != 0 {
			for _, id := range binds[int64(fc.TypePid)] {
				if name, ok := names[id]; ok {
					bc.Categories = append(bc.Categories, name)
					bc.Inherited = true
				}
			}
		}
		classes = append(classes, bc)
	}
	props := gonertia.Props{
		"source":     s,
		"classes":    classes,
		"list":       page.List,
		"paging":     gonertia.Props{"page": max(pg, 1), "pageCount": page.PageCount, "total": page.Total},
		"query":      gonertia.Props{"t": t, "wd": wd},
		"categories": options,
	}
	if err != nil {
		props["errors"] = formError(err.Error())
	}
	_ = inertia.RenderManage(c, "Collect/CollectBrowse", props)
}

// CollectBind 绑定/解除采集站分类与本站分类, 完成后返回浏览页
func CollectBind(c *gin.Context) {
	var req request.CollectBindRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.SourceId == "" || collect.Svc.GetFilmSource(req.SourceId) == nil {
		inertia.Redirect(c, "/manage/collect/list")
		return
	}
	_, names := categoryOptions()
	var ids []int64
	for _, id := range req.CategoryIds {
		if _, ok := names[id]; ok {
			ids = append(ids, id)
		}
	}
	_ = collect.Svc.Bind(req.SourceId, req.TypeId, req.TypeName, ids)
	inertia.Back(c)
}

// CollectBindClear 清空选中采集接口的分类绑定, 未选中时清空全部采集接口的绑定
func CollectBindClear(c *gin.Context) {
	var req request.IdsRequest[string]
	_ = c.ShouldBindJSON(&req)
	ids := req.Ids
	if len(ids) == 0 {
		for _, s := range collect.Svc.GetFilmSourceList() {
			ids = append(ids, s.Id)
		}
	}
	_ = collect.Svc.ClearBinds(ids...)
	inertia.Redirect(c, "/manage/collect/list")
}

// CollectFilmByIds 采集采集站中选中的影片
func CollectFilmByIds(c *gin.Context) {
	var req request.CollectFilmIdsRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Id == "" || len(req.Ids) == 0 {
		response.Failed("请求参数异常, 请选择需要采集的视频", c)
		return
	}
	ids := make([]string, len(req.Ids))
	for i, id := range req.Ids {
		ids[i] = strconv.FormatInt(id, 10)
	}
	if err := spider.Svc.CollectByIds(req.Id, strings.Join(ids, ",")); err != nil {
		response.Failed(err.Error(), c)
		return
	}
	response.SuccessOnlyMsg("采集任务已开启, 请稍后查看视频数据", c)
}

// CollectProgress 各采集接口最近一次的采集进度 (JSON, 后台采集接口列表轮询)
func CollectProgress(c *gin.Context) {
	response.Success(spider.Progresses(), "采集进度获取成功", c)
}
