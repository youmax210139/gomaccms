package admin

import (
	"fmt"
	"gomaccms/internal/http/request"
	"gomaccms/internal/http/response"
	"gomaccms/internal/http/session"
	"gomaccms/internal/spider"
	"gomaccms/internal/user"

	"github.com/gin-gonic/gin"
)

// StarSpider 开启并执行采集任务
func StarSpider(c *gin.Context) {
	var req request.SpiderStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Failed("请求参数异常!!!", c)
		return
	}
	if req.Time == 0 {
		response.Failed("采集开启失败,采集时长不能为0", c)
		return
	}
	if len(req.Id) <= 0 {
		response.Failed("采集开启失败, 采集接口Id为空", c)
		return
	}
	if err := spider.Svc.StartCollect(req.Id, req.Time); err != nil {
		response.Failed(fmt.Sprint("采集任务开启失败: ", err.Error()), c)
		return
	}
	response.SuccessOnlyMsg("采集任务已开始", c)
}

// StopSpider 中止采集接口正在进行的采集任务
func StopSpider(c *gin.Context) {
	var req request.SpiderStartRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Id == "" {
		response.Failed("请求参数异常", c)
		return
	}
	if err := spider.Svc.StopCollect(req.Id); err != nil {
		response.Failed(err.Error(), c)
		return
	}
	response.SuccessOnlyMsg("已请求中止, 当前页采集完后停止", c)
}

// ResumeSpider 从采集接口上次 (已中止 / 已中断 / 失败) 采集的中断处继续
func ResumeSpider(c *gin.Context) {
	var req request.SpiderStartRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Id == "" {
		response.Failed("请求参数异常", c)
		return
	}
	if err := spider.Svc.ResumeCollect(req.Id); err != nil {
		response.Failed(err.Error(), c)
		return
	}
	response.SuccessOnlyMsg("已从中断处继续采集", c)
}

// RetryFailedPages 重采一笔采集记录中失败的页
func RetryFailedPages(c *gin.Context) {
	var req struct {
		Id uint64 `json:"id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Id == 0 {
		response.Failed("请求参数异常", c)
		return
	}
	if err := spider.Svc.RetryFailedPages(req.Id); err != nil {
		response.Failed(err.Error(), c)
		return
	}
	response.SuccessOnlyMsg("已开始重试失败的页", c)
}

// SingleUpdateSpider 单一影片更新采集
func SingleUpdateSpider(c *gin.Context) {
	ids := c.Query("ids")
	if ids == "" {
		response.Failed("参数异常, 资源标识ID信息缺失", c)
		return
	}
	spider.Svc.SyncCollect(ids)
	response.SuccessOnlyMsg("视频更新任务已成功开启!!!", c)
}

// 校验密码有效性
func verifyPassword(c *gin.Context, password string) bool {
	uc, ok := session.Claims(c)
	if !ok {
		response.Failed("操作失败,登录信息异常!!!", c)
		return false
	}
	return user.Svc.VerifyUserPassword(uc.UserID, password)
}
