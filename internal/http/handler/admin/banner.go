package admin

import (
	"fmt"
	"strconv"

	"gomaccms/internal/config"
	"gomaccms/internal/film"
	"gomaccms/internal/http/request"
	"gomaccms/internal/http/response"
	"gomaccms/internal/i18n"
	"gomaccms/internal/paging"
	"gomaccms/internal/siteconfig"
	"gomaccms/internal/view/inertia"

	"github.com/gin-gonic/gin"
	gonertia "github.com/romsar/gonertia/v3"
)

// bannerScheme 海报管理当前的分类方案: 请求参数 scheme (或请求体中的 schemeId), 默认为默认方案
func bannerScheme(c *gin.Context, fromBody int64) int64 {
	if fromBody > 0 {
		return fromBody
	}
	if id, err := strconv.ParseInt(c.Query("scheme"), 10, 64); err == nil && film.CategorySvc.SchemeExists(id) {
		return id
	}
	return film.DefaultSchemeId
}

// schemeTranslations 分类方案启用的、原文以外的语言 (几个方案时取并集), 即需要编辑译文的语言
func schemeTranslations(schemeIds ...int64) []i18n.Language {
	langs := map[string]bool{}
	for _, id := range schemeIds {
		for _, code := range film.CategorySvc.Scheme(id).Langs {
			langs[code] = true
		}
	}
	var out []i18n.Language
	for _, l := range i18n.Svc.List() {
		if l.Enabled && l.Code != i18n.SourceLang && langs[l.Code] {
			out = append(out, l)
		}
	}
	return out
}

// renderBanners 渲染海报管理页 (分类方案的全部海报); errMsg 非空时作为错误提示
func renderBanners(c *gin.Context, schemeId int64, errMsg string) {
	banners := siteconfig.Svc.Banners(schemeId, false)
	var mids []int64
	for _, b := range banners {
		if b.Mid != 0 {
			mids = append(mids, b.Mid)
		}
	}
	_ = inertia.RenderManage(c, "System/Banners", withFormError(gonertia.Props{
		"scheme":  schemeId,
		"schemes": film.CategorySvc.ListSchemes(),
		"banners": banners,
		// 列表中显示绑定视频的名称
		"vodNames":  film.Svc.GetNamesByMids(mids),
		"languages": schemeTranslations(schemeId),
	}, errMsg))
}

// bannerDone 海报修改后清除该方案的首页缓存, 回到该方案的页签
func bannerDone(c *gin.Context, schemeId int64) {
	film.SearchRepo.RemoveCache(config.HomeCacheKeyOf(schemeId))
	inertia.Redirect(c, fmt.Sprintf("/manage/banner/list?scheme=%d", schemeId))
}

// BannerList 海报管理页(Inertia 渲染)
func BannerList(c *gin.Context) {
	renderBanners(c, bannerScheme(c, 0), "")
}

// BannerSave 新增 / 修改海报(Inertia 渲染)
func BannerSave(c *gin.Context) {
	var req request.BannerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		renderBanners(c, bannerScheme(c, 0), "海报参数异常")
		return
	}
	schemeId := bannerScheme(c, req.SchemeId)
	if req.Id > 0 {
		old, err := siteconfig.Svc.FindBanner(req.Id)
		if err != nil {
			renderBanners(c, schemeId, "海报不存在")
			return
		}
		schemeId = old.SchemeId
	}
	// 绑定的视频必须在该海报的分类方案中 (前台才看得到)
	if req.Mid != 0 {
		if _, _, ok := film.SearchRepo.SchemeCategoryOf(req.Mid, schemeId); !ok {
			renderBanners(c, schemeId, fmt.Sprintf("视频 %d 不在此分类方案中", req.Mid))
			return
		}
	}
	b := siteconfig.Banner{Id: req.Id, SchemeId: schemeId, Mid: req.Mid, Name: req.Name,
		Poster: req.Poster, Picture: req.Picture, Sort: req.Sort, Status: req.Status, I18n: req.I18n}
	if err := siteconfig.Svc.SaveBanner(&b); err != nil {
		renderBanners(c, schemeId, err.Error())
		return
	}
	bannerDone(c, schemeId)
}

// BannerState 启用 / 停用海报(Inertia 渲染)
func BannerState(c *gin.Context) {
	var req request.BannerRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Id == 0 {
		renderBanners(c, bannerScheme(c, 0), "海报参数异常")
		return
	}
	b, err := siteconfig.Svc.FindBanner(req.Id)
	if err != nil {
		renderBanners(c, bannerScheme(c, 0), "海报不存在")
		return
	}
	if err := siteconfig.Svc.SetBannerStatus(b.Id, req.Status); err != nil {
		renderBanners(c, b.SchemeId, err.Error())
		return
	}
	bannerDone(c, b.SchemeId)
}

// BannerDel 删除海报(Inertia 渲染)
func BannerDel(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Query("id"), 10, 64)
	b, err := siteconfig.Svc.FindBanner(id)
	if err != nil {
		renderBanners(c, bannerScheme(c, 0), "海报不存在")
		return
	}
	if err := siteconfig.Svc.DeleteBanner(b.Id); err != nil {
		renderBanners(c, b.SchemeId, err.Error())
		return
	}
	bannerDone(c, b.SchemeId)
}

// BannerVods 海报绑定视频的选择列表 (JSON): ?scheme= 分类方案, keyword 片名或视频ID, current 页码
func BannerVods(c *gin.Context) {
	schemeId := bannerScheme(c, 0)
	current, _ := strconv.Atoi(c.DefaultQuery("current", "1"))
	page := paging.Page{PageSize: 10, Current: max(current, 1)}
	list := film.SchemeVods(schemeId, c.Query("keyword"), &page)
	response.Success(gin.H{"list": list, "page": page}, "", c)
}
