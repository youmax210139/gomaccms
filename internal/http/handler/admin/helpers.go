package admin

import (
	"gomaccms/internal/paging"
	"gomaccms/internal/siteconfig"
	"net/url"
	"strconv"

	gonertia "github.com/romsar/gonertia/v3"
)

// maxPageSize 后台列表每页条数上限
const maxPageSize = 500

// pageFrom 从查询参数解析分页: current 无效时为 1, pageSize 无效或超出上限时使用后台设置的每页条数
func pageFrom(values url.Values) *paging.Page {
	p := &paging.Page{}
	if p.Current, _ = strconv.Atoi(values.Get("current")); p.Current <= 0 {
		p.Current = 1
	}
	if p.PageSize, _ = strconv.Atoi(values.Get("pageSize")); p.PageSize <= 0 || p.PageSize > maxPageSize {
		p.PageSize = siteconfig.Svc.AdminPageSize()
	}
	return p
}

// formError Inertia 页面的表单错误提示 (props.errors.form)
func formError(msg string) gonertia.Props {
	return gonertia.Props{"form": msg}
}

// withFormError msg 非空时在 props 中加入表单错误提示, 返回 props 本身
func withFormError(props gonertia.Props, msg string) gonertia.Props {
	if msg != "" {
		props["errors"] = formError(msg)
	}
	return props
}
