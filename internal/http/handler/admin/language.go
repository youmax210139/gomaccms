package admin

import (
	"gomaccms/internal/http/request"
	"gomaccms/internal/i18n"
	"gomaccms/internal/view/inertia"

	"github.com/gin-gonic/gin"
	gonertia "github.com/romsar/gonertia/v3"
)

// renderLanguages 渲染语言管理页; errMsg 非空时作为错误提示
func renderLanguages(c *gin.Context, errMsg string) {
	props := gonertia.Props{"languages": i18n.Svc.List(), "sourceLang": i18n.SourceLang}
	withFormError(props, errMsg)
	_ = inertia.RenderManage(c, "System/Languages", props)
}

// LanguageList 语言管理页(Inertia 渲染)
func LanguageList(c *gin.Context) {
	renderLanguages(c, "")
}

// LanguageSave 新增 / 修改语言(Inertia 渲染)
func LanguageSave(c *gin.Context) {
	var req request.LanguageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		renderLanguages(c, "请求参数异常")
		return
	}
	l := i18n.Language{Code: req.Code, Name: req.Name, LibreCode: req.LibreCode, Enabled: req.Enabled, Sort: req.Sort}
	if err := i18n.Svc.Save(l, req.IsNew); err != nil {
		renderLanguages(c, err.Error())
		return
	}
	inertia.Redirect(c, "/manage/lang/list")
}

// LanguageState 启用 / 停用语言(Inertia 渲染)
func LanguageState(c *gin.Context) {
	var req request.LanguageRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Code == "" {
		renderLanguages(c, "请求参数异常")
		return
	}
	if err := i18n.Svc.SetEnabled(req.Code, req.Enabled); err != nil {
		renderLanguages(c, err.Error())
		return
	}
	inertia.Redirect(c, "/manage/lang/list")
}
