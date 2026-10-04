package admin

import (
	"gomaccms/internal/film"
	"gomaccms/internal/http/response"
	"gomaccms/internal/publish"
	"gomaccms/internal/view/inertia"

	"github.com/gin-gonic/gin"
	gonertia "github.com/romsar/gonertia/v3"
)

// publishSchemeRequest 网站地图的操作对象: 分类方案
type publishSchemeRequest struct {
	SchemeId int64 `json:"schemeId"`
}

// bindScheme 读取请求中的分类方案, 不存在时返回 false 并回复错误
func bindScheme(c *gin.Context) (int64, bool) {
	var req publishSchemeRequest
	if err := c.ShouldBindJSON(&req); err != nil || !film.CategorySvc.SchemeExists(req.SchemeId) {
		response.Failed("分类方案不存在", c)
		return 0, false
	}
	return req.SchemeId, true
}

// PublishCenter 网站地图页面(Inertia 渲染); 状态由页面轮询 PublishStatus 更新
func PublishCenter(c *gin.Context) {
	_ = inertia.RenderManage(c, "System/Publish", gonertia.Props{"status": publish.Svc.Status(*pageFrom(c.Request.URL.Query()))})
}

// PublishStatus 各分类方案的 sitemap、RSS、IndexNow 状态与最近的任务
func PublishStatus(c *gin.Context) {
	response.Success(publish.Svc.Status(*pageFrom(c.Request.URL.Query())), "状态获取成功", c)
}

// PublishSitemap 开始重建分类方案 sitemap 的后台任务, 立即返回任务 (该方案已在重建时返回进行中的任务)
func PublishSitemap(c *gin.Context) {
	id, ok := bindScheme(c)
	if !ok {
		return
	}
	job, created, err := publish.Svc.RebuildSitemap(id)
	if err != nil {
		response.Failed("sitemap 任务建立失败: "+err.Error(), c)
		return
	}
	msg := "已开始重建 sitemap"
	if !created {
		msg = "该方案的 sitemap 正在重建中"
	}
	response.Success(gin.H{"job_id": job.Id, "status": job.Status, "job": job}, msg, c)
}

// PublishRSS 清除分类方案的 RSS 缓存, 下次请求重新生成
func PublishRSS(c *gin.Context) {
	id, ok := bindScheme(c)
	if !ok {
		return
	}
	publish.Svc.RefreshRSS(id)
	response.SuccessOnlyMsg("RSS 已刷新", c)
}

// PublishIndexNow 立即提交分类方案待推送的 IndexNow URL (后台任务)
func PublishIndexNow(c *gin.Context) {
	if !publish.Svc.IndexNow.Enabled() {
		response.Failed("IndexNow 未启用 (设置 INDEXNOW_ENABLED / INDEXNOW_KEY)", c)
		return
	}
	id, ok := bindScheme(c)
	if !ok {
		return
	}
	job, _, err := publish.Svc.SubmitIndexNow(id)
	if err != nil {
		response.Failed("IndexNow 任务建立失败: "+err.Error(), c)
		return
	}
	response.Success(gin.H{"job_id": job.Id, "status": job.Status, "job": job}, "已开始提交", c)
}

// PublishJob 任务状态
func PublishJob(c *gin.Context) {
	job, err := publish.Svc.Job(c.Param("id"))
	if err != nil {
		response.Failed("任务不存在", c)
		return
	}
	response.Success(job, "任务状态获取成功", c)
}

// CacheCenter 缓存管理页面(Inertia 渲染)
func CacheCenter(c *gin.Context) {
	_ = inertia.RenderManage(c, "System/Cache", gonertia.Props{})
}

// CacheRefresh 刷新缓存: home / today / rss / search / all
func CacheRefresh(c *gin.Context) {
	if err := publish.Svc.RefreshCache(c.Param("target")); err != nil {
		response.Failed(err.Error(), c)
		return
	}
	response.SuccessOnlyMsg("缓存已刷新", c)
}
