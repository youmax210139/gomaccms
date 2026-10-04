package admin

import (
	"errors"
	"fmt"
	"gomaccms/internal/cron"
	"gomaccms/internal/http/request"
	"gomaccms/internal/http/response"
	"gomaccms/internal/view/inertia"
	"strings"

	"github.com/gin-gonic/gin"
	gonertia "github.com/romsar/gonertia/v3"
)

// renderCron 渲染定时任务管理页; errMsg 非空时作为错误提示
func renderCron(c *gin.Context, errMsg string) {
	_ = inertia.RenderManage(c, "Cron/CronManage", withFormError(gonertia.Props{"list": cron.Svc.GetFilmCrontab()}, errMsg))
}

// FilmCronTaskList 定时任务管理页(Inertia 渲染)
func FilmCronTaskList(c *gin.Context) {
	renderCron(c, "")
}

// GetFilmCronTask 通过Id获取对应的定时任务信息
func GetFilmCronTask(c *gin.Context) {
	id := c.DefaultQuery("id", "")
	if id == "" {
		response.Failed("定时任务信息获取失败,任务Id不能为空", c)
		return
	}
	task, err := cron.Svc.GetFilmCrontabById(id)
	if err != nil {
		response.Failed(fmt.Sprint("定时任务信息获取失败", err.Error()), c)
		return
	}
	response.Success(task, "定时任务详情获取成功!!!", c)
}

// FilmCronAdd 添加定时任务(Inertia 渲染)
func FilmCronAdd(c *gin.Context) {
	var req request.FilmCronRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		renderCron(c, "请求参数异常!!!")
		return
	}
	vo := cron.FilmCronVo{Ids: req.Ids, Time: req.Time, Spec: req.Spec, Model: req.Model, State: req.State, Remark: req.Remark}
	if err := validTaskAddVo(vo); err != nil {
		renderCron(c, err.Error())
		return
	}
	vo.Spec = strings.TrimSpace(vo.Spec)
	if err := cron.Svc.AddFilmCrontab(vo); err != nil {
		renderCron(c, fmt.Sprint("定时任务添加失败: ", err.Error()))
		return
	}
	inertia.Redirect(c, "/manage/cron/list")
}

// FilmCronUpdate 更新定时任务信息(Inertia 渲染)
func FilmCronUpdate(c *gin.Context) {
	var req request.FilmCronTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		renderCron(c, "请求参数异常!!!")
		return
	}
	t := cron.FilmCollectTask{Id: req.Id, Ids: req.Ids, Time: req.Time, State: req.State, Remark: req.Remark}
	if err := validTaskInfo(t); err != nil {
		renderCron(c, err.Error())
		return
	}
	task, err := cron.Svc.GetFilmCrontabById(t.Id)
	if err != nil {
		renderCron(c, fmt.Sprint("更新失败: ", err.Error()))
		return
	}
	task.Ids = t.Ids
	task.Time = t.Time
	task.State = t.State
	task.Remark = t.Remark
	cron.Svc.UpdateFilmCron(task)
	inertia.Redirect(c, "/manage/cron/list")
}

// ChangeTaskState 开启 | 关闭Id 对应的定时任务(Inertia 渲染)
func ChangeTaskState(c *gin.Context) {
	var req request.FilmCronTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		renderCron(c, "请求参数异常!!!")
		return
	}
	task, err := cron.Svc.GetFilmCrontabById(req.Id)
	if err != nil {
		renderCron(c, fmt.Sprint("更新失败: ", err.Error()))
		return
	}
	task.State = req.State
	cron.Svc.UpdateFilmCron(task)
	inertia.Redirect(c, "/manage/cron/list")
}

// DelFilmCron 删除定时任务(Inertia 渲染)
func DelFilmCron(c *gin.Context) {
	id := c.DefaultQuery("id", "")
	if id != "" {
		_ = cron.Svc.DelFilmCrontab(id)
	}
	inertia.Redirect(c, "/manage/cron/list")
}

// -------------------------------------------------- 参数校验 --------------------------------------------------

func validTaskInfo(t cron.FilmCollectTask) error {
	if len(t.Id) <= 0 {
		return errors.New("参数校验失败, 任务Id信息不能为空")
	}
	if t.Time == 0 {
		return errors.New("参数校验失败, 采集时长不能为零值")
	}
	return nil
}

func validTaskAddVo(vo cron.FilmCronVo) error {
	switch vo.Model {
	case 0:
		if vo.Time == 0 {
			return errors.New("参数校验失败, 采集时长不能为零值")
		}
	case 1:
		if vo.Time == 0 {
			return errors.New("参数校验失败, 采集时长不能为零值")
		}
		if len(vo.Ids) == 0 {
			return errors.New("参数校验失败, 自定义更新未绑定任何资源站点")
		}
	default:
		return errors.New("参数校验失败, 未定义的任务类型")
	}
	if err := cron.ValidSpec(vo.Spec); err != nil {
		return errors.New(fmt.Sprint("参数校验失败 cron表达式校验失败: ", err.Error()))
	}
	return nil
}
