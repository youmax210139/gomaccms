package public

import (
	"net/http"

	"gomaccms/internal/film"
	"gomaccms/internal/member"

	"github.com/gin-gonic/gin"
)

// guestDenied 前台尚无会员登录, 访客一律套用游客组的分类权限; 无权限时渲染主题的 notice 页 (403) 并返回 true,
// 提示文字由 notice 页用语言包 notice.denied 与 perm.<权限> 组成
func guestDenied(c *gin.Context, perm string, categoryIds ...int64) bool {
	if member.Svc.GuestAllows(perm, categoryIds...) {
		return false
	}
	c.Status(http.StatusForbidden)
	RenderPage(c, "notice", map[string]any{"perm": perm})
	return true
}

// guestNav 去掉游客没有列表页权限的导航分类
func guestNav(nav []*film.Category) []*film.Category {
	g := member.Svc.GuestGroup()
	out := make([]*film.Category, 0, len(nav))
	for _, c := range nav {
		if g.Allows(c.Id, member.PermList) {
			out = append(out, c)
		}
	}
	return out
}
