package admin

import (
	"fmt"
	"gomaccms/internal/collect"
	"gomaccms/internal/film"
	"gomaccms/internal/http/request"
	"gomaccms/internal/http/response"
	"gomaccms/internal/spider"
	"gomaccms/internal/view/inertia"

	"github.com/gin-gonic/gin"
	gonertia "github.com/romsar/gonertia/v3"
)

func toFilmSource(req request.FilmSourceRequest) collect.FilmSource {
	s := collect.FilmSource{
		Id: req.Id, Name: req.Name, Uri: req.Uri, Params: req.Params,
		ResultModel: collect.CollectResultModel(req.ResultModel),
		CollectType: collect.ResourceType(req.CollectType), Operation: collect.DataOperation(req.Operation),
		FilterMode: collect.FilterMode(req.FilterMode), FilterCode: req.FilterCode, FilterYear: req.FilterYear,
		SyncImage: collect.SyncImageMode(req.SyncImage), State: req.State, Interval: req.Interval,
	}
	collect.NormalizeFilmSource(&s)
	return s
}

// FilmSourceList 采集接口列表页(Inertia 渲染)
func FilmSourceList(c *gin.Context) {
	renderCollectList(c, "")
}

// renderCollectList 渲染采集接口列表页; errMsg 非空时作为错误提示
func renderCollectList(c *gin.Context, errMsg string) {
	_ = inertia.RenderManage(c, "Collect/CollectManage", withFormError(gonertia.Props{"list": collect.Svc.GetFilmSourceList()}, errMsg))
}

// FindFilmSource 通过ID返回对应的资源站数据
func FindFilmSource(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		response.Failed("参数异常, 资源站标识不能为空", c)
		return
	}
	fs := collect.Svc.GetFilmSource(id)
	if fs == nil {
		response.Failed("数据异常,资源站信息不存在", c)
		return
	}
	response.Success(fs, "原站点详情信息查找成功", c)
}

// renderFilmSourceForm 渲染采集接口表单页, msg 非空时作为表单错误提示
func renderFilmSourceForm(c *gin.Context, s collect.FilmSource, msg string) {
	props := gonertia.Props{"source": s}
	withFormError(props, msg)
	_ = inertia.RenderManage(c, "Collect/CollectForm", props)
}

// FilmSourceAddPage 新增采集接口页(Inertia 渲染), 默认 json / 视频 / 新增+更新 / 不过滤 / 图片跟随全局
func FilmSourceAddPage(c *gin.Context) {
	renderFilmSourceForm(c, collect.FilmSource{State: true}, "")
}

// FilmSourceEditPage 编辑采集接口页(Inertia 渲染)
func FilmSourceEditPage(c *gin.Context) {
	fs := collect.Svc.GetFilmSource(c.Query("id"))
	if fs == nil {
		inertia.Redirect(c, "/manage/collect/list")
		return
	}
	renderFilmSourceForm(c, *fs, "")
}

// FilmSourceAdd 保存新增的采集接口, 成功后返回采集接口列表
func FilmSourceAdd(c *gin.Context) {
	var req request.FilmSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		renderFilmSourceForm(c, collect.FilmSource{}, "请求参数异常")
		return
	}
	s := toFilmSource(req)
	s.Id = ""
	if err := collect.Svc.ValidFilmSource(s); err != nil {
		renderFilmSourceForm(c, s, err.Error())
		return
	}
	if err := collect.Svc.SaveFilmSource(s); err != nil {
		renderFilmSourceForm(c, s, fmt.Sprint("采集接口添加失败: ", err.Error()))
		return
	}
	inertia.Redirect(c, "/manage/collect/list")
}

// FilmSourceUpdate 保存采集接口修改, 成功后返回采集接口列表
func FilmSourceUpdate(c *gin.Context) {
	var req request.FilmSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		renderFilmSourceForm(c, collect.FilmSource{}, "请求参数异常")
		return
	}
	s := toFilmSource(req)
	if s.Id == "" || collect.Svc.GetFilmSource(s.Id) == nil {
		renderFilmSourceForm(c, s, "数据异常, 采集接口信息不存在")
		return
	}
	if err := collect.Svc.ValidFilmSource(s); err != nil {
		renderFilmSourceForm(c, s, err.Error())
		return
	}
	if err := collect.Svc.UpdateFilmSource(s); err != nil {
		renderFilmSourceForm(c, s, fmt.Sprint("采集接口更新失败: ", err.Error()))
		return
	}
	inertia.Redirect(c, "/manage/collect/list")
}

// FilmSourceChange 采集接口启用/停用(Inertia 渲染), 完成后返回列表
func FilmSourceChange(c *gin.Context) {
	var req request.FilmSourceStateRequest
	if err := c.ShouldBindJSON(&req); err == nil && req.Id != "" {
		if err = collect.Svc.ChangeFilmSourceState(req.Id, req.State); err != nil {
			renderCollectList(c, err.Error())
			return
		}
	}
	inertia.Redirect(c, "/manage/collect/list")
}

// FilmSourceDel 采集接口删除(Inertia 渲染), GET ?id= 删除单个, POST {ids} 批量删除; 主站点不会被删除
func FilmSourceDel(c *gin.Context) {
	var ids []string
	if c.Request.Method == "POST" {
		var req request.IdsRequest[string]
		_ = c.ShouldBindJSON(&req)
		ids = req.Ids
	} else if id := c.Query("id"); id != "" {
		ids = []string{id}
	}
	var msg string
	for _, id := range ids {
		if err := collect.Svc.DelFilmSource(id); err != nil {
			msg = err.Error()
		}
	}
	if msg != "" {
		renderCollectList(c, msg)
		return
	}
	inertia.Redirect(c, "/manage/collect/list")
}

// FilmSourceTest 测试影视站点数据是否可用
func FilmSourceTest(c *gin.Context) {
	var req request.FilmSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Failed("请求参数异常", c)
		return
	}
	s := toFilmSource(req)
	if err := collect.Svc.ValidFilmSource(s); err != nil {
		response.Failed(err.Error(), c)
		return
	}
	if err := spider.CollectApiTest(s); err != nil {
		response.Failed(err.Error(), c)
		return
	}
	response.SuccessOnlyMsg("测试成功!!!", c)
}

// GetNormalFilmSource 获取状态为启用的采集站信息
func GetNormalFilmSource(c *gin.Context) {
	var l []collect.FilmTaskOptions
	for _, v := range collect.Svc.GetFilmSourceList() {
		if v.State {
			l = append(l, collect.FilmTaskOptions{Id: v.Id, Name: v.Name})
		}
	}
	response.Success(l, "影视源信息获取成功", c)
}

// CollectClearPreview 清空采集接口视频前的统计: 将删除的视频数 (只来自该接口) 与保留的共用视频数
func CollectClearPreview(c *gin.Context) {
	id := c.Query("id")
	if collect.Svc.GetFilmSource(id) == nil {
		response.Failed("采集接口不存在", c)
		return
	}
	exclusive, shared := film.SourceVodCounts(id)
	response.Success(gin.H{"exclusive": exclusive, "shared": shared, "collecting": spider.IsCollecting(id)}, "统计成功", c)
}

// CollectClear 清空采集接口的全部视频: 只来自该接口的视频整部删除, 与其他接口共用的视频只移除该接口的播放线路
func CollectClear(c *gin.Context) {
	var req request.CollectClearRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Id == "" {
		response.Failed("请求参数异常", c)
		return
	}
	s := collect.Svc.GetFilmSource(req.Id)
	if s == nil {
		response.Failed("采集接口不存在", c)
		return
	}
	if req.Password == "" || !verifyPassword(c, req.Password) {
		response.Failed("清空失败, 密码校验失败", c)
		return
	}
	if spider.IsCollecting(s.Id) {
		response.Failed("该采集接口正在采集中, 请等待采集完成后再清空", c)
		return
	}
	deleted, stripped, err := film.ClearSourceVods(s.Id)
	spider.ClearCache()
	if err != nil {
		response.Failed(fmt.Sprintf("清空未完成 (已删除 %d 部, 已移除 %d 部的播放线路): %v", deleted, stripped, err), c)
		return
	}
	response.SuccessOnlyMsg(fmt.Sprintf("已清空「%s」的视频: 删除 %d 部, %d 部与其他采集接口共用的视频已移除该接口的播放线路", s.Name, deleted, stripped), c)
}

// CollectLogList 采集接口最近的采集历史 (JSON)
func CollectLogList(c *gin.Context) {
	response.Success(collect.Repo.CollectLogs(c.Query("id"), 20), "采集记录获取成功", c)
}
