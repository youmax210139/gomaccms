package admin

import (
	"gomaccms/internal/film"
	"gomaccms/internal/http/request"
	"gomaccms/internal/view/inertia"

	"github.com/gin-gonic/gin"
	gonertia "github.com/romsar/gonertia/v3"
)

// renderPlayers 渲染播放器管理页; errMsg 非空时作为错误提示
func renderPlayers(c *gin.Context, errMsg string) {
	props := gonertia.Props{
		"players": film.ListPlayers(),
		"counts":  film.PlayerFilmCounts(),
	}
	withFormError(props, errMsg)
	_ = inertia.RenderManage(c, "Vod/Player", props)
}

// PlayerList 播放器管理页(Inertia 渲染)
func PlayerList(c *gin.Context) {
	renderPlayers(c, "")
}

// PlayerSave 新增 / 修改播放器(Inertia 渲染)
func PlayerSave(c *gin.Context) {
	var req request.PlayerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		renderPlayers(c, "请求参数异常")
		return
	}
	p := film.Player{Code: req.Code, Name: req.Name, Status: req.Status, Sort: req.Sort, Remark: req.Remark}
	if err := film.SavePlayer(&p, req.IsNew); err != nil {
		renderPlayers(c, err.Error())
		return
	}
	inertia.Redirect(c, "/manage/film/player")
}

// PlayerState 启用 / 停用播放器(Inertia 渲染)
func PlayerState(c *gin.Context) {
	var req request.PlayerRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Code == "" {
		renderPlayers(c, "请求参数异常")
		return
	}
	if err := film.SetPlayerStatus(req.Code, req.Status); err != nil {
		renderPlayers(c, err.Error())
		return
	}
	inertia.Redirect(c, "/manage/film/player")
}

// PlayerDel 删除播放器(Inertia 渲染)
func PlayerDel(c *gin.Context) {
	if code := c.Query("code"); code != "" {
		_ = film.DeletePlayer(code)
	}
	inertia.Redirect(c, "/manage/film/player")
}
