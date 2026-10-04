// Package tags MacCMS 模板数据标签的实现: 每个标签一个 Handler, 只调用 film 等服务 / 仓库 (不写 SQL);
// 结果按 标签 + 参数 + 方案 + 语言 缓存在 Redis (沿用 db.Rdb, 失败时直接查询)
package tags

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gomaccms/internal/ad"
	"gomaccms/internal/db"
	"gomaccms/internal/film"
	"gomaccms/internal/publish"
	"gomaccms/internal/view/maccms"
)

// cacheTTL 标签结果的缓存时间 (影片变动后最多这么久才在 MacCMS 标签中更新)
const cacheTTL = 2 * time.Minute

// Register 注册全部数据标签 (bootstrap.Wire 调用); 新标签在这里加一行
func Register() {
	maccms.Register("vod", cached("vod", vod))
	maccms.Register("type", cached("type", typeTag))
	// 文章、专题、友情链接: 本站还没有这些数据, 先注册为空列表, 模板可以照常使用
	for _, name := range []string{"art", "topic", "link"} {
		maccms.Register(name, empty)
	}
	// 筛选: 对应分类 (type) 的检索标签
	for name, title := range map[string]string{"class": "Plot", "tag": "Plot", "area": "Area", "lang": "Language", "year": "Year"} {
		maccms.Register(name, cached(name, filter(title)))
	}
	maccms.Register("letter", letters)
	// 广告不缓存: 后台修改后立即生效 (按索引查询, 很快)
	maccms.Register("ad", adTag)
	// 版本、状态: 本站没有这些检索标签
	maccms.Register("version", empty)
	maccms.Register("state", empty)
}

// cached 以 Redis 缓存 Handler 的结果
func cached(name string, h maccms.Handler) maccms.Handler {
	return func(ctx maccms.Ctx, o maccms.QueryOptions) (maccms.Result, error) {
		key := fmt.Sprintf("MacCMS:Tag:%s:%x", name, sha1.Sum([]byte(fmt.Sprintf("%d|%s|%d|%s|%+v", ctx.SchemeId, ctx.Lang, ctx.TypeId, ctx.BaseURL, o))))
		if db.Rdb != nil {
			if s, err := db.Rdb.Get(db.Cxt, key).Result(); err == nil {
				var r maccms.Result
				if json.Unmarshal([]byte(s), &r) == nil {
					return r, nil
				}
			}
		}
		r, err := h(ctx, o)
		if err == nil && db.Rdb != nil {
			if b, e := json.Marshal(r); e == nil {
				db.Rdb.Set(db.Cxt, key, b, cacheTTL)
			}
		}
		return r, err
	}
}

func empty(maccms.Ctx, maccms.QueryOptions) (maccms.Result, error) {
	return maccms.Result{}, nil
}

func scheme(ctx maccms.Ctx) int64 {
	if ctx.SchemeId <= 0 {
		return film.DefaultSchemeId
	}
	return ctx.SchemeId
}

// typeIds 标签的 type 参数: current 为当前页面的分类, ids 为指定分类, all 为不限
func typeIds(ctx maccms.Ctx, o maccms.QueryOptions) []int64 {
	switch o.Type {
	case "current":
		if ctx.TypeId > 0 {
			return []int64{ctx.TypeId}
		}
	case "ids":
		return o.TypeIds
	}
	return nil
}

// vod {maccms:vod}: 方案中可见的影片 (按请求语言显示); version 本站没有对应栏位, 不筛选
func vod(ctx maccms.Ctx, o maccms.QueryOptions) (maccms.Result, error) {
	class := o.Class
	if class == "" {
		class = o.Tag
	}
	vl, total := film.QueryVods(film.VodQuery{
		SchemeId: scheme(ctx), Ids: o.Ids, TypeIds: typeIds(ctx, o), Class: class, Area: o.Area, Lang: o.Lang,
		Year: o.Year, Letter: o.Letter, State: o.State, OrderBy: o.By, Desc: o.Order == "desc",
		Offset: o.Offset, Limit: o.Num,
	}, o.Paging)
	film.LocalizeVods(vl, ctx.Lang)
	list := make([]maccms.Item, 0, len(vl))
	for _, v := range vl {
		list = append(list, maccms.Item{
			"vod_id": strconv.FormatInt(v.VodId, 10), "vod_name": v.VodName, "vod_sub": v.VodSub, "vod_en": v.VodEn,
			"vod_pic": v.VodPic, "vod_url": publish.VodPath(v.VodId), "vod_remarks": v.VodRemarks, "vod_actor": v.VodActor,
			"vod_director": v.VodDirector, "vod_area": v.VodArea, "vod_lang": v.VodLang, "vod_year": v.VodYear,
			"vod_class": v.VodClass, "vod_letter": v.VodLetter, "vod_state": v.VodState,
			"vod_score": strconv.FormatFloat(v.VodScore, 'f', 1, 64), "vod_hits": strconv.FormatInt(v.VodHits, 10),
			"vod_time": time.Unix(v.VodTime, 0).Format(time.DateTime),
			"type_id":  strconv.FormatInt(v.TypeId, 10), "type_id_1": strconv.FormatInt(v.TypeId1, 10), "type_name": v.TypeName,
		})
	}
	return maccms.Result{List: list, Total: int(total)}, nil
}

// typeTag {maccms:type}: 方案中显示的视频分类 (按请求语言); type 为上级分类时列出其子分类, 默认为一级分类;
// 按 sort 由小到大 (by="id" 时按 ID 与 order)
func typeTag(ctx maccms.Ctx, o maccms.QueryOptions) (maccms.Result, error) {
	tree := film.CategoryRepo.GetSchemeCategoryTree(scheme(ctx))
	tree.Localize(ctx.Lang)
	parents := typeIds(ctx, o)
	var all []*film.CategoryTree
	for _, top := range tree.Children {
		if !top.Show {
			continue
		}
		if len(o.Ids) > 0 { // 指定分类 (一级或二级)
			for _, c := range append([]*film.CategoryTree{top}, top.Children...) {
				if c.Show && containsId(o.Ids, c.Id) {
					all = append(all, c)
				}
			}
			continue
		}
		if len(parents) == 0 {
			all = append(all, top)
		} else if containsId(parents, top.Id) {
			for _, c := range top.Children {
				if c.Show {
					all = append(all, c)
				}
			}
		}
	}
	if o.By == "id" {
		sortById(all, o.Order == "desc")
	}
	total := len(all)
	all = all[min(o.Offset, total):]
	all = all[:min(o.Num, len(all))]
	list := make([]maccms.Item, 0, len(all))
	for _, c := range all {
		u := publish.ClassifyPath(c.Id)
		if c.Pid != 0 {
			u = fmt.Sprintf("/filmClassifySearch?Pid=%d&Category=%d", c.Pid, c.Id)
		}
		list = append(list, maccms.Item{
			"type_id": strconv.FormatInt(c.Id, 10), "type_pid": strconv.FormatInt(c.Pid, 10), "type_name": c.Name,
			"type_en": c.Slug, "type_sort": strconv.FormatInt(c.Sort, 10), "type_url": u,
		})
	}
	return maccms.Result{List: list, Total: total}, nil
}

func containsId(ids []int64, id int64) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

func sortById(l []*film.CategoryTree, desc bool) {
	for i := 1; i < len(l); i++ {
		for j := i; j > 0 && (l[j].Id < l[j-1].Id) != desc; j-- {
			l[j], l[j-1] = l[j-1], l[j]
		}
	}
}

// filterParam 筛选标签对应的影片库网址参数
var filterParam = map[string]string{"Plot": "Plot", "Area": "Area", "Language": "Language", "Year": "Year"}

// filter {maccms:class} / {maccms:area} ...: type 指定 (或当前页面) 的一级分类的检索标签, 每项 name / value / url
func filter(title string) maccms.Handler {
	return func(ctx maccms.Ctx, o maccms.QueryOptions) (maccms.Result, error) {
		ids := typeIds(ctx, o)
		if len(ids) == 0 {
			return maccms.Result{}, nil
		}
		pid := ids[0]
		var list []maccms.Item
		for _, t := range film.SearchRepo.GetTagsByTitle(pid, title) {
			name, value, ok := strings.Cut(t, ":")
			if !ok {
				name, value = t, t
			}
			list = append(list, maccms.Item{"name": name, "value": value,
				"url": fmt.Sprintf("/filmClassifySearch?Pid=%d&%s=%s", pid, filterParam[title], url.QueryEscape(value))})
		}
		total := len(list)
		list = list[min(o.Offset, total):]
		return maccms.Result{List: list[:min(o.Num, len(list))], Total: total}, nil
	}
}

// letters {maccms:letter}: A-Z 与 0-9
func letters(ctx maccms.Ctx, o maccms.QueryOptions) (maccms.Result, error) {
	var list []maccms.Item
	for _, r := range "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789" {
		list = append(list, maccms.Item{"name": string(r), "value": string(r)})
	}
	total := len(list)
	list = list[min(o.Offset, total):]
	return maccms.Result{List: list[:min(o.Num, len(list))], Total: total}, nil
}

// adTag {maccms:ad slot="play_top"}: 方案中该广告位启用的广告, 每项 ad_id / ad_name / ad_pic / ad_link / ad_slot
func adTag(ctx maccms.Ctx, o maccms.QueryOptions) (maccms.Result, error) {
	if o.Slot == "" {
		return maccms.Result{}, fmt.Errorf("缺少 slot 参数")
	}
	l, total := ad.Svc.Slot(scheme(ctx), o.Slot, o.Offset, o.Num)
	list := make([]maccms.Item, 0, len(l))
	for _, a := range l {
		list = append(list, maccms.Item{"ad_id": strconv.FormatInt(a.Id, 10), "ad_name": a.Name, "ad_pic": a.Image,
			"ad_link": a.Link, "ad_slot": a.Slot})
	}
	return maccms.Result{List: list, Total: int(total)}, nil
}
