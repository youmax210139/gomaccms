package admin

import (
	"gomaccms/internal/http/middleware"
	"gomaccms/internal/http/request"
	"gomaccms/internal/http/session"
	"gomaccms/internal/user"
	"gomaccms/internal/view/inertia"
	"slices"

	"github.com/gin-gonic/gin"
	gonertia "github.com/romsar/gonertia/v3"
)

const adminListPath = "/manage/admin/list"

// renderAdmins 渲染管理员管理页; errMsg 非空时作为错误提示
func renderAdmins(c *gin.Context, errMsg string) {
	founder, _ := user.Svc.Access(session.UserID(c))
	props := gonertia.Props{
		"admins":      user.Svc.ListAdmins(),
		"permissions": middleware.Permissions,
		"founderId":   user.Repo.FounderId(),
		"isFounder":   founder,
	}
	withFormError(props, errMsg)
	_ = inertia.RenderManage(c, "User/Admins", props)
}

// AdminList 管理员管理页(Inertia 渲染)
func AdminList(c *gin.Context) {
	renderAdmins(c, "")
}

// AdminSave 新增 / 修改管理员(Inertia 渲染)
func AdminSave(c *gin.Context) {
	var req struct {
		Id          uint     `json:"id"`
		UserName    string   `json:"userName"`
		Password    string   `json:"password"`
		NickName    string   `json:"nickName"`
		Email       string   `json:"email"`
		Status      int      `json:"status"`
		Permissions []string `json:"permissions"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		renderAdmins(c, "请求参数异常")
		return
	}
	// 只保留有效的权限 key; 授权其他管理员时不能超出自己的权限
	founder, own := user.Svc.Access(session.UserID(c))
	if !founder && req.Id != 0 && req.Id == user.Repo.FounderId() {
		renderAdmins(c, "只有创始管理员本人可以修改创始管理员")
		return
	}
	perms := []string{}
	for _, k := range middleware.AllPermissionKeys() {
		if slices.Contains(req.Permissions, k) && (founder || slices.Contains(own, k)) {
			perms = append(perms, k)
		}
	}
	in := user.AdminInput{Id: req.Id, UserName: req.UserName, Password: req.Password, NickName: req.NickName,
		Email: req.Email, Status: req.Status, Permissions: perms}
	if !founder && req.Id != 0 {
		// 非创始管理员修改其他管理员时, 保留对方自己权限范围外的已有权限
		for _, k := range user.Repo.FindById(req.Id).Permissions {
			if !slices.Contains(own, k) && !slices.Contains(in.Permissions, k) {
				in.Permissions = append(in.Permissions, k)
			}
		}
	}
	if err := user.Svc.SaveAdmin(in, session.UserID(c)); err != nil {
		renderAdmins(c, err.Error())
		return
	}
	inertia.Redirect(c, adminListPath)
}

// AdminState 启用 / 停用管理员(Inertia 渲染)
func AdminState(c *gin.Context) {
	var req request.StatusRequest[uint, int]
	if err := c.ShouldBindJSON(&req); err != nil || req.Id == 0 {
		renderAdmins(c, "请求参数异常")
		return
	}
	if err := user.Svc.SetAdminStatus(req.Id, req.Status, session.UserID(c)); err != nil {
		renderAdmins(c, err.Error())
		return
	}
	inertia.Redirect(c, adminListPath)
}

// AdminDel 删除管理员 (可批量)(Inertia 渲染)
func AdminDel(c *gin.Context) {
	var req request.IdsRequest[uint]
	if err := c.ShouldBindJSON(&req); err != nil {
		renderAdmins(c, "请求参数异常")
		return
	}
	if err := user.Svc.DeleteAdmins(req.Ids, session.UserID(c)); err != nil {
		renderAdmins(c, err.Error())
		return
	}
	inertia.Redirect(c, adminListPath)
}
