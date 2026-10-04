package publish

import (
	"fmt"
	"gomaccms/internal/film"
	"gomaccms/internal/member"
)

// 站内页面路径: sitemap、RSS、IndexNow 与页面的 canonical 共用, 保证一致
const (
	HomePath  = "/index"
	TodayPath = "/today"
	RankPath  = "/rank"
)

// VodPath 影片详情页路径
func VodPath(id int64) string {
	return fmt.Sprintf("/filmDetail?link=%d", id)
}

// ClassifyPath 一级分类页路径
func ClassifyPath(pid int64) string {
	return fmt.Sprintf("/filmClassify?Pid=%d", pid)
}

// detailTypeIds 分类方案中游客可以看内容页的视频分类: 显示中的一级分类, 以及一级、二级都显示且有权限的二级分类
// (与前台 public.guestDenied 对内容页的判断一致: 一级与二级分类都要允许)
func detailTypeIds(schemeId int64) []int64 {
	var ids []int64
	for _, c := range film.CategoryRepo.GetSchemeCategoryTree(schemeId).Children {
		if !c.Show || !member.Svc.GuestAllows(member.PermDetail, c.Id) {
			continue
		}
		ids = append(ids, c.Id)
		for _, sub := range c.Children {
			if sub.Show && member.Svc.GuestAllows(member.PermDetail, c.Id, sub.Id) {
				ids = append(ids, sub.Id)
			}
		}
	}
	return ids
}

// listPids 分类方案中游客可以看列表页的一级分类 (导航中显示的分类)
func listPids(schemeId int64) []int64 {
	var ids []int64
	for _, c := range film.CategoryRepo.GetSchemeCategoryTree(schemeId).Children {
		if c.Show && member.Svc.GuestAllows(member.PermList, c.Id) {
			ids = append(ids, c.Id)
		}
	}
	return ids
}
