package admin

import (
	"fmt"
	"gomaccms/internal/ad"
	"gomaccms/internal/film"
	"gomaccms/internal/http/request"
	"gomaccms/internal/view/inertia"
	"strconv"

	"github.com/gin-gonic/gin"
	gonertia "github.com/romsar/gonertia/v3"
)

// renderAds 渲染广告管理页 (分类方案的全部广告); errMsg 非空时作为错误提示
func renderAds(c *gin.Context, schemeId int64, errMsg string) {
	_ = inertia.RenderManage(c, "System/Ads", withFormError(gonertia.Props{
		"scheme":  schemeId,
		"schemes": film.CategorySvc.ListSchemes(),
		"ads":     ad.Svc.List(schemeId),
		"slots":   ad.Slots,
	}, errMsg))
}

// adDone 回到该方案的页签
func adDone(c *gin.Context, schemeId int64) {
	inertia.Redirect(c, fmt.Sprintf("/manage/ad/list?scheme=%d", schemeId))
}

// AdList 广告管理页 (Inertia 渲染), ?scheme= 选择分类方案
func AdList(c *gin.Context) {
	renderAds(c, bannerScheme(c, 0), "")
}

// AdSave 新增 / 修改广告 (Inertia 渲染); 修改时分类方案不变
func AdSave(c *gin.Context) {
	var req request.AdRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		renderAds(c, bannerScheme(c, 0), "广告参数异常")
		return
	}
	schemeId := bannerScheme(c, req.SchemeId)
	if req.Id > 0 {
		old, err := ad.Svc.Find(req.Id)
		if err != nil {
			renderAds(c, schemeId, "广告不存在")
			return
		}
		schemeId = old.SchemeId
	}
	a := ad.Ad{Id: req.Id, SchemeId: schemeId, Slot: req.Slot, Name: req.Name, Image: req.Image, Link: req.Link,
		Sort: req.Sort, Status: req.Status}
	if err := ad.Svc.Save(&a); err != nil {
		renderAds(c, schemeId, err.Error())
		return
	}
	adDone(c, schemeId)
}

// AdState 启用 / 停用广告 (Inertia 渲染)
func AdState(c *gin.Context) {
	var req request.AdRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Id == 0 {
		renderAds(c, bannerScheme(c, 0), "广告参数异常")
		return
	}
	a, err := ad.Svc.Find(req.Id)
	if err != nil {
		renderAds(c, bannerScheme(c, 0), "广告不存在")
		return
	}
	if err := ad.Svc.SetStatus(a.Id, req.Status); err != nil {
		renderAds(c, a.SchemeId, err.Error())
		return
	}
	adDone(c, a.SchemeId)
}

// AdDel 删除广告 (Inertia 渲染)
func AdDel(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Query("id"), 10, 64)
	a, err := ad.Svc.Find(id)
	if err != nil {
		renderAds(c, bannerScheme(c, 0), "广告不存在")
		return
	}
	if err := ad.Svc.Delete(a.Id); err != nil {
		renderAds(c, a.SchemeId, err.Error())
		return
	}
	adDone(c, a.SchemeId)
}
