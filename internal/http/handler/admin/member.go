package admin

import (
	"gomaccms/internal/film"
	"gomaccms/internal/http/request"
	"gomaccms/internal/member"
	"gomaccms/internal/view/inertia"
	"net/url"
	"strconv"

	"github.com/gin-gonic/gin"
	gonertia "github.com/romsar/gonertia/v3"
)

const memberGroupListPath = "/manage/member/group/list"

// permCategory 会员组权限表格中的一行分类
type permCategory struct {
	Id    int64  `json:"id"`
	Name  string `json:"name"`
	Level int    `json:"level"`
}

// permScheme 一个分类方案的视频分类 (按树的顺序展开)
type permScheme struct {
	Id         int64          `json:"id"`
	Name       string         `json:"name"`
	Categories []permCategory `json:"categories"`
}

func permSchemes() []permScheme {
	var l []permScheme
	for _, s := range film.CategorySvc.ListSchemes() {
		ps := permScheme{Id: s.Id, Name: s.Name}
		var walk func(nodes []*film.CategoryTree, level int)
		walk = func(nodes []*film.CategoryTree, level int) {
			for _, n := range nodes {
				ps.Categories = append(ps.Categories, permCategory{Id: n.Id, Name: n.Name, Level: level})
				walk(n.Children, level+1)
			}
		}
		walk(film.CategoryRepo.GetSchemeCategoryTree(s.Id).Children, 0)
		l = append(l, ps)
	}
	return l
}

// renderMemberGroups 渲染会员组管理页; errMsg 非空时作为错误提示
func renderMemberGroups(c *gin.Context, errMsg string) {
	props := gonertia.Props{
		"groups":  member.Svc.ListGroups(),
		"schemes": permSchemes(),
		"perms":   member.AllPerms,
	}
	withFormError(props, errMsg)
	_ = inertia.RenderManage(c, "User/MemberGroups", props)
}

// MemberGroupList 会员组管理页(Inertia 渲染)
func MemberGroupList(c *gin.Context) {
	renderMemberGroups(c, "")
}

// MemberGroupSave 新增 / 修改会员组(Inertia 渲染)
func MemberGroupSave(c *gin.Context) {
	var g member.Group
	if err := c.ShouldBindJSON(&g); err != nil {
		renderMemberGroups(c, "请求参数异常")
		return
	}
	if err := member.Svc.SaveGroup(g); err != nil {
		renderMemberGroups(c, err.Error())
		return
	}
	inertia.Redirect(c, memberGroupListPath)
}

// MemberGroupState 启用 / 停用会员组(Inertia 渲染)
func MemberGroupState(c *gin.Context) {
	var req request.StatusRequest[int64, bool]
	if err := c.ShouldBindJSON(&req); err != nil {
		renderMemberGroups(c, "请求参数异常")
		return
	}
	if err := member.Svc.SetGroupStatus(req.Id, req.Status); err != nil {
		renderMemberGroups(c, err.Error())
		return
	}
	inertia.Redirect(c, memberGroupListPath)
}

// MemberGroupDel 删除会员组 (可批量)(Inertia 渲染)
func MemberGroupDel(c *gin.Context) {
	var req request.IdsRequest[int64]
	if err := c.ShouldBindJSON(&req); err != nil {
		renderMemberGroups(c, "请求参数异常")
		return
	}
	if err := member.Svc.DeleteGroups(req.Ids); err != nil {
		renderMemberGroups(c, err.Error())
		return
	}
	inertia.Redirect(c, memberGroupListPath)
}

// memberQuery 从查询参数解析会员列表的筛选条件
func memberQuery(values url.Values) member.MemberQuery {
	q := member.MemberQuery{Status: values.Get("status"), Keyword: values.Get("keyword"), Paging: pageFrom(values)}
	q.GroupId, _ = strconv.ParseInt(values.Get("groupId"), 10, 64)
	return q
}

// renderMembers 渲染会员管理页; errMsg 非空时作为错误提示
func renderMembers(c *gin.Context, q member.MemberQuery, errMsg string) {
	props := gonertia.Props{
		"members": member.Svc.ListMembers(q),
		"groups":  member.Svc.ListGroups(),
		"query":   q,
	}
	withFormError(props, errMsg)
	_ = inertia.RenderManage(c, "User/Members", props)
}

// MemberList 会员管理页(Inertia 渲染)
func MemberList(c *gin.Context) {
	renderMembers(c, memberQuery(c.Request.URL.Query()), "")
}

// memberBack 会员操作后回到操作前的列表 (保留筛选与页码)
func memberBack(c *gin.Context, err error) {
	ref, _ := url.Parse(c.GetHeader("Referer"))
	values := url.Values{}
	if ref != nil {
		values = ref.Query()
	}
	if err != nil {
		renderMembers(c, memberQuery(values), err.Error())
		return
	}
	target := "/manage/member/list"
	if len(values) > 0 {
		target += "?" + values.Encode()
	}
	inertia.Redirect(c, target)
}

// MemberSave 新增 / 修改会员(Inertia 渲染)
func MemberSave(c *gin.Context) {
	var in member.MemberInput
	if err := c.ShouldBindJSON(&in); err != nil {
		memberBack(c, err)
		return
	}
	memberBack(c, member.Svc.SaveMember(in))
}

// MemberState 启用 / 停用会员(Inertia 渲染)
func MemberState(c *gin.Context) {
	var req request.StatusRequest[int64, bool]
	if err := c.ShouldBindJSON(&req); err != nil {
		memberBack(c, err)
		return
	}
	memberBack(c, member.Svc.SetMemberStatus(req.Id, req.Status))
}

// MemberDel 删除会员 (可批量)(Inertia 渲染)
func MemberDel(c *gin.Context) {
	var req request.IdsRequest[int64]
	if err := c.ShouldBindJSON(&req); err != nil {
		memberBack(c, err)
		return
	}
	memberBack(c, member.Svc.DeleteMembers(req.Ids))
}
