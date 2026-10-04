package film

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"gomaccms/internal/config"
	"gomaccms/internal/db"
	"gomaccms/internal/paging"
	"gomaccms/internal/util"
	"log"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var SearchRepo *SearchRepository

type SearchRepository struct{}

func NewSearchRepository() *SearchRepository {
	return &SearchRepository{}
}

// visibleVod 只查询显示中的影片 (所属分类未被隐藏)
func visibleVod() *gorm.DB {
	return db.Mdb.Model(&Vod{}).Where("vod_status = ?", vodStatusOn)
}

// vodInCategory 属于某分类 (column 为 type_id 或 type_id_1) 且该分类未停用的影片
func vodInCategory(column string, id int64) *gorm.DB {
	return visibleVod().Where("vod_id IN (?)",
		db.Mdb.Model(&VodType{}).Select("vod_id").Where(column+" = ? AND status = ?", id, vodStatusOn))
}

// vodInScheme 在某分类方案中有 (未停用的) 分类的影片, 即该方案对应的站点可以看到的影片
func vodInScheme(schemeId int64) *gorm.DB {
	return db.Mdb.Model(&VodType{}).Select("vod_id").Where("scheme_id = ? AND status = ?", schemeId, vodStatusOn)
}

// vodSortColumns 检索标签中的排序值 (沿用旧 search 表的列名, 已缓存在 Redis 中) → vod 表栏位
var vodSortColumns = map[string]string{
	"update_stamp":  "vod_time",
	"hits":          "vod_hits",
	"score":         "vod_score",
	"release_stamp": "vod_time_add",
}

func (r *SearchRepository) SaveSearchTag(search SearchInfo) {
	key := fmt.Sprintf(config.SearchTitle, search.Pid)
	searchMap := db.Rdb.HGetAll(db.Cxt, key).Val()
	if len(searchMap) == 0 {
		searchMap = make(map[string]string)
		searchMap["Category"] = "类型"
		searchMap["Plot"] = "剧情"
		searchMap["Area"] = "地区"
		searchMap["Language"] = "语言"
		searchMap["Year"] = "年份"
		searchMap["Initial"] = "首字母"
		searchMap["Sort"] = "排序"
		db.Rdb.HMSet(db.Cxt, key, searchMap)
	}
	for k := range searchMap {
		tagKey := fmt.Sprintf(config.SearchTag, search.Pid, k)
		tagCount := db.Rdb.ZCard(db.Cxt, tagKey).Val()
		switch k {
		case "Category":
			if tagCount == 0 {
				for _, t := range CategoryRepo.GetChildrenTree(search.Pid) {
					db.Rdb.ZAdd(db.Cxt, fmt.Sprintf(config.SearchTag, search.Pid, k),
						redis.Z{Score: float64(-t.Id), Member: fmt.Sprintf("%v:%v", t.Name, t.Id)})
				}
			}
		case "Year":
			if tagCount == 0 {
				currentYear := time.Now().Year()
				for i := 0; i < 12; i++ {
					db.Rdb.ZAdd(db.Cxt, fmt.Sprintf(config.SearchTag, search.Pid, k),
						redis.Z{Score: float64(currentYear - i), Member: fmt.Sprintf("%v:%v", currentYear-i, currentYear-i)})
				}
			}
		case "Initial":
			if tagCount == 0 {
				for i := 65; i <= 90; i++ {
					db.Rdb.ZAdd(db.Cxt, fmt.Sprintf(config.SearchTag, search.Pid, k),
						redis.Z{Score: float64(90 - i), Member: fmt.Sprintf("%c:%c", i, i)})
				}
			}
		case "Sort":
			if tagCount == 0 {
				tags := []redis.Z{
					{Score: 3, Member: "时间排序:update_stamp"},
					{Score: 2, Member: "人气排序:hits"},
					{Score: 1, Member: "评分排序:score"},
					{Score: 0, Member: "最新上映:release_stamp"},
				}
				db.Rdb.ZAdd(db.Cxt, fmt.Sprintf(config.SearchTag, search.Pid, k), tags...)
			}
		case "Plot":
			handleSearchTags(search.ClassTag, tagKey)
		case "Area":
			handleSearchTags(search.Area, tagKey)
		case "Language":
			handleSearchTags(search.Language, tagKey)
		}
	}
}

// RebuildMissingTags 为有影片却没有检索标签 (Search:Pid<id>:Title) 的一级视频分类补建标签,
// 修复之前直接归入一级分类的影片把标签都记到了 Pid0 下; 已有标签的分类不处理, 每次启动执行
func (r *SearchRepository) RebuildMissingTags() {
	for _, c := range CategoryRepo.List(CategoryVideo, 0) {
		if c.Pid != 0 || db.Rdb.Exists(db.Cxt, fmt.Sprintf(config.SearchTitle, c.Id)).Val() == 1 {
			continue
		}
		var vl []Vod
		ids := db.Mdb.Model(&VodType{}).Select("vod_id").Where("type_id_1 = ?", c.Id)
		if err := db.Mdb.Select(vodBasicColumns).Where("vod_id IN (?)", ids).Find(&vl).Error; err != nil {
			log.Println("RebuildMissingTags Error:", err)
			continue
		}
		for _, v := range vl {
			info := v.Search()
			info.Pid = c.Id
			r.SaveSearchTag(info)
		}
		if len(vl) > 0 {
			log.Printf("分类「%s」补建检索标签: %d 部影片", c.Name, len(vl))
		}
	}
}

// handleSearchTags 对应原本 model/system/Search.go 里导出的 HandleSearchTags,
// 只有 SaveSearchTag 内部会用到,改成 unexported。
func handleSearchTags(preTags string, k string) {
	preTags = regexp.MustCompile(`[\s\n\r]+`).ReplaceAllString(preTags, "")
	f := func(sep string) {
		for _, t := range strings.Split(preTags, sep) {
			score := db.Rdb.ZScore(db.Cxt, k, fmt.Sprintf("%v:%v", t, t)).Val()
			db.Rdb.ZAdd(db.Cxt, k, redis.Z{Score: score + 1, Member: fmt.Sprintf("%v:%v", t, t)})
		}
	}
	switch {
	case strings.Contains(preTags, "/"):
		f("/")
	case strings.Contains(preTags, ","):
		f(",")
	case strings.Contains(preTags, "，"):
		f("，")
	case strings.Contains(preTags, "、"):
		f("、")
	default:
		if len(preTags) == 0 {
			// 无 tag 信息则不缓存(原样保留)
		} else if preTags == "其它" {
			db.Rdb.ZAdd(db.Cxt, k, redis.Z{Score: 0, Member: fmt.Sprintf("%v:%v", preTags, preTags)})
		} else {
			score := db.Rdb.ZScore(db.Cxt, k, fmt.Sprintf("%v:%v", preTags, preTags)).Val()
			db.Rdb.ZAdd(db.Cxt, k, redis.Z{Score: score + 1, Member: fmt.Sprintf("%v:%v", preTags, preTags)})
		}
	}
}

// UpsertVods 按影片ID新增或更新影片 (后台手动添加等, 采集入库见 IngestVods): 新影片整行写入; 已存在的只更新播放地址、更新状态等随采集变化的栏位
// (名称、主要分类等可能被后台修改过, 不覆盖). 影片的分类 (vod_type) 只补上新增的, 已有的不删除
func UpsertVods(ml []MovieDetail) error {
	if len(ml) == 0 {
		return nil
	}
	vl := make([]Vod, 0, len(ml))
	for _, m := range ml {
		vl = append(vl, NewVod(m))
	}
	err := db.Mdb.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "vod_id"}},
		DoUpdates: clause.AssignmentColumns(vodUpdateColumns),
	}).CreateInBatches(&vl, 200).Error
	if err != nil {
		return err
	}
	// Mid 为 0 的影片 (后台手动添加) 由数据库分配ID
	ids := make([]int64, len(ml))
	for i := range ml {
		ml[i].Mid = vl[i].VodId
		ids[i] = vl[i].VodId
	}
	if err := addVodTypes(ml); err != nil {
		return err
	}
	VodsChanged(ids, vodSchemes(ids))
	return nil
}

// addVodTypes 按影片的 Types (没有时为 Cid) 补上 vod_type 记录, 新增的分类同时登记检索标签
func addVodTypes(ml []MovieDetail) error {
	var ids, typeIds []int64
	for _, m := range ml {
		ids = append(ids, m.Mid)
		typeIds = append(typeIds, append(m.Types, m.Cid)...)
	}
	var cl []Category
	db.Mdb.Where("id IN ?", typeIds).Find(&cl)
	categories := make(map[int64]Category, len(cl))
	for _, c := range cl {
		categories[c.Id] = c
	}
	var existing []VodType
	db.Mdb.Select("vod_id", "type_id").Where("vod_id IN ?", ids).Find(&existing)
	exists := make(map[[2]int64]bool, len(existing))
	for _, e := range existing {
		exists[[2]int64{e.VodId, e.TypeId}] = true
	}
	var rows []VodType
	for _, m := range ml {
		types := m.Types
		if len(types) == 0 && m.Cid != 0 {
			types = []int64{m.Cid}
		}
		for _, t := range types {
			if exists[[2]int64{m.Mid, t}] {
				continue
			}
			exists[[2]int64{m.Mid, t}] = true
			// 分类不在分类表中 (如采集站分类尚未导入) 时沿用影片自身的 pid, 归入默认方案
			row := VodType{VodId: m.Mid, TypeId: t, TypeId1: m.Pid, SchemeId: DefaultSchemeId, Status: vodStatusOn}
			if c, ok := categories[t]; ok {
				// 直接归入一级分类时 pid 即分类自身 (与 spider.applyCategoryBind 一致), 按 pid 查询的列表、计数才找得到
				row.TypeId1, row.SchemeId = c.Pid, c.SchemeId
				if c.Pid == 0 {
					row.TypeId1 = c.Id
				}
				if !c.Show {
					row.Status = vodStatusOff
				}
			}
			rows = append(rows, row)
			info := NewVod(m).Search()
			info.Cid, info.Pid = row.TypeId, row.TypeId1
			SearchRepo.SaveSearchTag(info)
		}
	}
	if len(rows) == 0 {
		return nil
	}
	return db.Mdb.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(&rows, 500).Error
}

// findVods 按条件查询影片摘要
func findVods(query *gorm.DB) []Vod {
	var vl []Vod
	if err := query.Select(vodBasicColumns).Find(&vl).Error; err != nil {
		log.Println("Find Vod Error:", err)
		return nil
	}
	return vl
}

func toSearchInfos(vl []Vod) []SearchInfo {
	sl := make([]SearchInfo, 0, len(vl))
	for _, v := range vl {
		sl = append(sl, v.Search())
	}
	return sl
}

// 关键字搜索的结果缓存在 vod_search 表, 过期后重新查询; 每个关键字最多缓存 vodSearchMaxIds 个结果
const (
	vodSearchField  = "vod_name|vod_sub"
	vodSearchTTL    = int64(3600)
	vodSearchMaxIds = 1000
)

// SearchFilmKeyword 按片名/别名搜索影片, 返回当前页的影片ID; 只包含 schemeId 分类方案中的影片
func (r *SearchRepository) SearchFilmKeyword(keyword string, page *paging.Page, schemeId int64, lang string) []int64 {
	ids := filterInScheme(r.keywordResultIds(keyword, lang), schemeId)
	page.Total = len(ids)
	page.PageCount = (page.Total + page.PageSize - 1) / page.PageSize
	start := (page.Current - 1) * page.PageSize
	if start < 0 || start >= len(ids) {
		return nil
	}
	return ids[start:min(start+page.PageSize, len(ids))]
}

// vodSearchFieldOf vod_search 的搜索栏位: 原文语言保持旧值 (已有缓存与热搜不变), 其他语言加上语言代码
func vodSearchFieldOf(lang string) string {
	if !translated(lang) {
		return vodSearchField
	}
	return vodSearchField + "|" + lang
}

// keywordQuery 关键字匹配的前台可见影片ID: 原文片名/别名; 非原文语言再比对该语言的译名
func keywordQuery(tx *gorm.DB, word, lang string) *gorm.DB {
	q := tx.Model(&Vod{}).Select("vod_id").Where("vod_status = ?", vodStatusOn)
	if !translated(lang) {
		return q.Where(textMatch(word, true, "vod_name", "vod_sub"))
	}
	return q.Where("(? OR vod_id IN (?))", textMatch(word, true, "vod_name", "vod_sub"),
		tx.Model(&VodI18n{}).Select("vod_id").Where("lang = ?", lang).Where(textMatch(word, true, "name", "sub")))
}

// keywordResultIds 关键字的全部结果ID: 优先读取 vod_search 缓存, 并累计命中次数
func (r *SearchRepository) keywordResultIds(keyword, lang string) []int64 {
	word := clip(keyword, 128)
	if word == "" {
		return nil
	}
	now := time.Now().Unix()
	field := vodSearchFieldOf(lang)
	key := fmt.Sprintf("%x", md5.Sum([]byte(word+"|"+field)))
	var cache VodSearch
	found := db.Mdb.Where("search_key = ?", key).Limit(1).Find(&cache).RowsAffected > 0
	if found && now-cache.SearchUpdateTime < vodSearchTTL {
		db.Mdb.Model(&VodSearch{}).Where("search_key = ?", key).Updates(map[string]any{
			"search_hit_count": gorm.Expr("search_hit_count + 1"), "search_last_hit_time": now})
		return parseIds(cache.SearchResultIds)
	}
	var ids []int64
	keywordQuery(db.Mdb, word, lang).Order("vod_year DESC, vod_time DESC").Limit(vodSearchMaxIds).Pluck("vod_id", &ids)
	strIds := make([]string, len(ids))
	for i, id := range ids {
		strIds[i] = strconv.FormatInt(id, 10)
	}
	cache = VodSearch{SearchKey: key, SearchWord: word, SearchField: field, SearchHitCount: cache.SearchHitCount + 1,
		SearchLastHitTime: now, SearchUpdateTime: now, SearchResultCount: len(ids), SearchResultIds: strings.Join(strIds, ",")}
	if err := db.Mdb.Save(&cache).Error; err != nil {
		log.Println("Save VodSearch Error:", err)
	}
	return ids
}

// HotKeywords 热门搜索词: 搜索次数最多且有结果的关键字, 不足 limit 个时用 pids 一级分类中人气最高的片名补足
func (r *SearchRepository) HotKeywords(pids []int64, limit int, lang string) []string {
	var words []string
	// 每个语言各自的热搜词
	db.Mdb.Model(&VodSearch{}).Where("search_result_count > 0 AND search_field = ?", vodSearchFieldOf(lang)).
		Order("search_hit_count DESC, search_last_hit_time DESC").Limit(limit).Pluck("search_word", &words)
	if len(words) >= limit || len(pids) == 0 {
		return words
	}
	var top []Vod
	vodInPids(pids).Select("vod_id", "vod_name").Order("vod_hits DESC").Limit(limit * 2).Find(&top)
	LocalizeVods(top, lang)
	for _, v := range top {
		if len(words) >= limit {
			break
		}
		if v.VodName != "" && !slices.Contains(words, v.VodName) {
			words = append(words, v.VodName)
		}
	}
	return words
}

// filterInScheme 只保留在 schemeId 分类方案中有分类的影片, 保持原有顺序
func filterInScheme(ids []int64, schemeId int64) []int64 {
	if len(ids) == 0 {
		return ids
	}
	var keep []int64
	db.Mdb.Model(&Vod{}).Where("vod_id IN ?", ids).Where("vod_id IN (?)", vodInScheme(schemeId)).Pluck("vod_id", &keep)
	kept := make(map[int64]bool, len(keep))
	for _, id := range keep {
		kept[id] = true
	}
	l := ids[:0:0]
	for _, id := range ids {
		if kept[id] {
			l = append(l, id)
		}
	}
	return l
}

func parseIds(s string) []int64 {
	var ids []int64
	for _, v := range strings.Split(s, ",") {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

func (r *SearchRepository) GetSearchTag(pid int64) map[string]interface{} {
	res := make(map[string]interface{})
	titles := db.Rdb.HGetAll(db.Cxt, fmt.Sprintf(config.SearchTitle, pid)).Val()
	res["titles"] = titles
	// 前台影片库只提供 类型 / 剧情 / 排序 三行筛选 (地区、语言、年份参数仍可用, 只是不显示)
	sortList := []string{"Category", "Plot", "Sort"}
	tagMap := make(map[string]interface{})
	for _, t := range sortList {
		tagMap[t] = handleTagStr(t, r.GetTagsByTitle(pid, t)...)
	}
	res["tags"] = tagMap
	res["sortList"] = sortList
	return res
}

func (r *SearchRepository) GetTagsByTitle(pid int64, t string) []string {
	var tags []string
	switch t {
	case "Category":
		for _, c := range CategoryRepo.GetChildrenTree(pid) {
			if c.Show {
				tags = append(tags, fmt.Sprintf("%s:%d", c.Name, c.Id))
			}
		}
	case "Plot":
		tags = topTags(fmt.Sprintf(config.SearchTag, pid, t), 40)
	case "Area":
		tags = topTags(fmt.Sprintf(config.SearchTag, pid, t), 11)
	case "Language":
		tags = topTags(fmt.Sprintf(config.SearchTag, pid, t), 6)
	case "Year", "Initial":
		tags = topTags(fmt.Sprintf(config.SearchTag, pid, t), -1)
	case "Sort":
		// 与分类首页的三个区块一致 (Redis 中旧的「时间排序」等选项不再显示)
		tags = []string{"最新上映:release_stamp", "最多浏览:hits", "评分最高:score"}
	}
	return tags
}

// handleTagStr 对应原本导出的 HandleTagStr,只在本档内使用,改成 unexported。
func handleTagStr(title string, tags ...string) []map[string]string {
	var r []map[string]string
	if !strings.EqualFold(title, "Sort") {
		r = append(r, map[string]string{"Name": "全部", "Value": ""})
	}
	for _, t := range tags {
		if sl := strings.Split(t, ":"); len(sl) > 0 {
			r = append(r, map[string]string{"Name": sl[0], "Value": sl[1]})
		}
	}
	if !strings.EqualFold(title, "Sort") && !strings.EqualFold(title, "Year") && !strings.EqualFold(title, "Category") {
		r = append(r, map[string]string{"Name": "其它", "Value": "其它"})
	}
	return r
}

func (r *SearchRepository) GetSearchInfosByTags(st SearchTagsVO, page *paging.Page) []int64 {
	qw := visibleVod().Select("vod_id")
	t := reflect.TypeOf(st)
	v := reflect.ValueOf(st)
	for i := 0; i < t.NumField(); i++ {
		value := v.Field(i).Interface()
		if util.IsEmpty(value) {
			continue
		}
		var ts []string
		if vv, flag := value.(string); flag && strings.EqualFold(vv, "其它") {
			for _, tagStr := range r.GetTagsByTitle(st.Pid, t.Field(i).Name) {
				ts = append(ts, strings.Split(tagStr, ":")[1])
			}
		}
		switch strings.ToLower(t.Field(i).Name) {
		case "pid":
			qw = qw.Where("vod_id IN (?)", db.Mdb.Model(&VodType{}).Select("vod_id").Where("type_id_1 = ? AND status = ?", value, vodStatusOn))
		case "cid":
			qw = qw.Where("vod_id IN (?)", db.Mdb.Model(&VodType{}).Select("vod_id").Where("type_id = ? AND status = ?", value, vodStatusOn))
		case "year":
			qw = qw.Where("vod_year = ?", fmt.Sprint(value))
		case "area", "language":
			column := map[string]string{"area": "vod_area", "language": "vod_lang"}[strings.ToLower(t.Field(i).Name)]
			if strings.EqualFold(value.(string), "其它") {
				qw = qw.Where(column+" NOT IN ?", ts)
				break
			}
			qw = qw.Where(column+" = ?", value)
		case "plot":
			if strings.EqualFold(value.(string), "其它") {
				for _, tagStr := range ts {
					qw = qw.Where("vod_class NOT LIKE ?", fmt.Sprintf("%%%v%%", tagStr))
				}
				break
			}
			qw = qw.Where(textMatch(fmt.Sprint(value), true, "vod_class"))
		case "sort":
			// 排序值来自请求参数, 只接受白名单内的栏位
			column, ok := vodSortColumns[fmt.Sprint(value)]
			if !ok {
				break
			}
			if column == "vod_time_add" {
				qw = qw.Order("vod_year DESC, vod_time_add DESC")
				break
			}
			qw = qw.Order(column + " DESC")
		}
	}
	paging.Apply(qw, page)
	var ids []int64
	if err := paging.Limit(qw, page).Pluck("vod_id", &ids).Error; err != nil {
		log.Println(err)
		return nil
	}
	return ids
}

// GetMovieListBySort 一级分类下的影片: t = 0 最新上映 (添加时间) / 1 最多浏览 / 2 评分最高
func (r *SearchRepository) GetMovieListBySort(t int, pid int64, page *paging.Page) []MovieBasicInfo {
	var ids []int64
	qw := paging.Limit(vodInCategory("type_id_1", pid).Select("vod_id"), page)
	switch t {
	case 0:
		qw = qw.Order("vod_time_add DESC")
	case 1:
		qw = qw.Order("vod_hits DESC")
	case 2:
		qw = qw.Order("vod_score DESC, vod_hits DESC")
	}
	if err := qw.Pluck("vod_id", &ids).Error; err != nil {
		log.Println(err)
		return nil
	}
	return MovieRepo.GetBasicInfoByIds(ids)
}

// vodInPids 属于这些一级分类 (且分类未停用) 的可见影片
func vodInPids(pids []int64) *gorm.DB {
	return visibleVod().Where("vod_id IN (?)",
		db.Mdb.Model(&VodType{}).Select("vod_id").Where("type_id_1 IN ? AND status = ?", pids, vodStatusOn))
}

// todayStart 今天 0 点 (本地时间) 的时间戳
func todayStart() int64 {
	y, m, d := time.Now().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.Local).Unix()
}

// TodayMovies 今天更新的影片 (新的在前), 只含 pids 一级分类中的影片
func (r *SearchRepository) TodayMovies(pids []int64, page *paging.Page) []MovieBasicInfo {
	if len(pids) == 0 {
		return nil
	}
	qw := vodInPids(pids).Where("vod_time >= ?", todayStart())
	paging.Apply(qw, page)
	var ids []int64
	if err := paging.Limit(qw, page).Order("vod_time DESC").Pluck("vod_id", &ids).Error; err != nil {
		log.Println("TodayMovies Error:", err)
		return nil
	}
	return MovieRepo.GetBasicInfoByIds(ids)
}

// TodayCount 今天更新的影片数, 只含 pids 一级分类中的影片
func (r *SearchRepository) TodayCount(pids []int64) int64 {
	var n int64
	if len(pids) > 0 {
		vodInPids(pids).Where("vod_time >= ?", todayStart()).Count(&n)
	}
	return n
}

// rankOrders 排行榜的排序方式
var rankOrders = map[string]string{
	"hits":  "vod_hits DESC, vod_time DESC",
	"score": "vod_score DESC, vod_hits DESC",
	"new":   "vod_time_add DESC",
}

// RankMovies 一级分类的排行 (by: hits 人气 / score 评分 / new 新上架), 最多 limit 部
func (r *SearchRepository) RankMovies(pid int64, by string, limit int) []SearchInfo {
	order, ok := rankOrders[by]
	if !ok {
		order = rankOrders["hits"]
	}
	return toSearchInfos(findVods(vodInCategory("type_id_1", pid).Order(order).Limit(limit)))
}

func (r *SearchRepository) GetSearchPage(s SearchVo) []SearchInfo {
	// 后台列表包含未审核的视频, 按审核 / 推荐 / 锁定筛选
	query := db.Mdb.Model(&Vod{})
	for column, value := range map[string]string{"vod_status": s.Status, "vod_level": s.Level, "vod_lock": s.Lock} {
		if v, err := strconv.Atoi(value); err == nil {
			query = query.Where(column+" = ?", v)
		}
	}
	if s.Name != "" {
		query = query.Where(textMatch(s.Name, true, "vod_name", "vod_sub"))
	}
	switch {
	case s.Cid > 0:
		query = query.Where("vod_id IN (?)", db.Mdb.Model(&VodType{}).Select("vod_id").Where("type_id = ?", s.Cid))
	case s.Pid > 0:
		query = query.Where("vod_id IN (?)", db.Mdb.Model(&VodType{}).Select("vod_id").Where("type_id_1 = ?", s.Pid))
	case s.SchemeId > 0:
		query = query.Where("vod_id IN (?)", db.Mdb.Model(&VodType{}).Select("vod_id").Where("scheme_id = ?", s.SchemeId))
	}
	if s.Plot != "" {
		query = query.Where(textMatch(s.Plot, false, "vod_class"))
	}
	if s.Area != "" {
		query = query.Where("vod_area = ?", s.Area)
	}
	if s.Language != "" {
		query = query.Where("vod_lang = ?", s.Language)
	}
	if int(s.Year) > time.Now().Year()-12 {
		query = query.Where("vod_year = ?", strconv.FormatInt(s.Year, 10))
	}
	switch s.Remarks {
	case "完结":
		query = query.Where("vod_remarks IN ?", []string{"完结", "HD"})
	case "":
	default:
		query = query.Where("vod_remarks NOT IN ?", []string{"完结", "HD"})
	}
	if s.Player != "" {
		query = query.Where("FIND_IN_SET(?, REPLACE(vod_play_from, '$$$', ','))", s.Player)
	}
	switch s.Picture {
	case "has":
		query = query.Where("vod_pic <> ''")
	case "none":
		query = query.Where("vod_pic = ''")
	}
	if s.BeginTime > 0 {
		query = query.Where("vod_time >= ?", s.BeginTime)
	}
	if s.EndTime > 0 {
		query = query.Where("vod_time <= ?", s.EndTime)
	}
	paging.Apply(query, s.Paging)
	sortColumn := map[string]string{"add": "vod_time_add", "hits": "vod_hits", "score": "vod_score", "id": "vod_id"}[s.Sort]
	if sortColumn == "" {
		sortColumn = "vod_time"
	}
	query = paging.Limit(query.Order(sortColumn+" DESC"), s.Paging)
	return toSearchInfos(findVods(query))
}

func (r *SearchRepository) GetSearchOptions(pid int64) map[string]interface{} {
	titles := db.Rdb.HGetAll(db.Cxt, fmt.Sprintf(config.SearchTitle, pid)).Val()
	tagMap := make(map[string]interface{})
	for t := range titles {
		switch t {
		case "Plot", "Area", "Language", "Year":
			tagMap[t] = handleTagStr(t, r.GetTagsByTitle(pid, t)...)
		}
	}
	return tagMap
}

// GetSearchInfoById 按影片ID (vod_id) 查询
func (r *SearchRepository) GetSearchInfoById(id int64) *SearchInfo {
	vl := findVods(db.Mdb.Model(&Vod{}).Where("vod_id = ?", id).Limit(1))
	if len(vl) == 0 {
		return nil
	}
	s := vl[0].Search()
	return &s
}

// DelFilmSearch 删除影片 (vod 记录及其分类、来源)
func (r *SearchRepository) DelFilmSearch(id int64) error {
	schemes := vodSchemes([]int64{id})
	err := db.Mdb.Transaction(func(tx *gorm.DB) error {
		if err := deleteVodRelations(tx, id); err != nil {
			return err
		}
		return tx.Where("vod_id = ?", id).Delete(&Vod{}).Error
	})
	if err != nil {
		log.Println(err)
		return err
	}
	VodsChanged([]int64{id}, schemes)
	return nil
}

// ShieldFilmSearch 停用分类: 该分类下的影片不再出现在这个分类中 (影片在其他分类中不受影响)
func (r *SearchRepository) ShieldFilmSearch(cid int64) error {
	defer VodsChanged(nil, nil)
	return db.Mdb.Model(&VodType{}).Where("type_id = ?", cid).Update("status", vodStatusOff).Error
}

// RecoverFilmSearch 恢复启用分类
func (r *SearchRepository) RecoverFilmSearch(cid int64) error {
	defer VodsChanged(nil, nil)
	return db.Mdb.Model(&VodType{}).Where("type_id = ?", cid).Update("status", vodStatusOn).Error
}

// CountByCategory 统计影片数 (含停用分类中的影片), 分别按 cid 与 pid 分组
func (r *SearchRepository) CountByCategory() (byCid, byPid map[int64]int64) {
	group := func(column string) map[int64]int64 {
		var rows []struct {
			Id    int64
			Total int64
		}
		if err := db.Mdb.Model(&VodType{}).Select(column + " AS id, COUNT(DISTINCT vod_id) AS total").Group(column).Scan(&rows).Error; err != nil {
			log.Println("CountByCategory Error:", err)
		}
		m := make(map[int64]int64, len(rows))
		for _, row := range rows {
			m[row.Id] = row.Total
		}
		return m
	}
	return group("type_id"), group("type_id_1")
}

// TransferCategory 将 column (cid 或 pid) = from 的影片改为分类 to; 已在 to 中的影片只去掉原分类
func (r *SearchRepository) TransferCategory(column string, from int64, to Category) error {
	defer VodsChanged(nil, nil)
	col := map[string]string{"cid": "type_id", "pid": "type_id_1"}[column]
	if col == "" {
		return fmt.Errorf("unknown category column %q", column)
	}
	return db.Mdb.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("UPDATE IGNORE vod_type SET type_id = ?, type_id_1 = ?, scheme_id = ? WHERE "+col+" = ?",
			to.Id, to.Pid, to.SchemeId, from).Error; err != nil {
			return err
		}
		if err := tx.Where(col+" = ?", from).Delete(&VodType{}).Error; err != nil {
			return err
		}
		return tx.Model(&Vod{}).Where(col+" = ?", from).
			Updates(map[string]any{"type_id": to.Id, "type_id_1": to.Pid, "type_name": to.Name}).Error
	})
}

// SchemeCategoryOf 影片在某分类方案中的分类 (第一个未停用的), 不在该方案中时 ok 为 false
func (r *SearchRepository) SchemeCategoryOf(vodId, schemeId int64) (cid, pid int64, ok bool) {
	var vt VodType
	res := db.Mdb.Where("vod_id = ? AND scheme_id = ? AND status = ?", vodId, schemeId, vodStatusOn).Order("type_id").Limit(1).Find(&vt)
	if res.Error != nil || res.RowsAffected == 0 {
		return 0, 0, false
	}
	return vt.TypeId, vt.TypeId1, true
}

func (r *SearchRepository) DataCache(key string, data map[string]interface{}) {
	val, _ := json.Marshal(data)
	db.Rdb.Set(db.Cxt, key, val, time.Minute*30)
}

func (r *SearchRepository) GetCacheData(key string) map[string]interface{} {
	data := make(map[string]interface{})
	val, err := db.Rdb.Get(db.Cxt, key).Result()
	if err != nil || len(val) <= 0 {
		return nil
	}
	_ = json.Unmarshal([]byte(val), &data)
	return data
}

// RemoveCache 删除缓存 key 以及按域名区分的 key:* (如各域名的首页缓存)
func (r *SearchRepository) RemoveCache(key string) {
	keys := append(db.Rdb.Keys(db.Cxt, key+":*").Val(), key)
	db.Rdb.Del(db.Cxt, keys...)
}

// GetNamesByMids 按影片 mid 批量查询影片名称 (图库中影片封面的来源说明)
func (r *SearchRepository) GetNamesByMids(mids []int64) map[int64]string {
	names := make(map[int64]string, len(mids))
	if len(mids) == 0 {
		return names
	}
	var vl []Vod
	if err := db.Mdb.Model(&Vod{}).Select("vod_id, vod_name").Where("vod_id IN ?", mids).Find(&vl).Error; err != nil {
		log.Println("GetNamesByMids Error:", err)
	}
	for _, v := range vl {
		names[v.VodId] = v.VodName
	}
	return names
}

// vodBatchColumns 后台批量设置允许修改的栏位及取值范围
var vodBatchColumns = map[string]struct {
	column   string
	min, max int64
}{
	"level":  {"vod_level", 0, 9},
	"status": {"vod_status", 0, 1},
	"lock":   {"vod_lock", 0, 1},
	"hits":   {"vod_hits", 0, 1 << 40},
}

// BatchUpdateVods 批量设置视频的推荐 / 审核 / 锁定 / 人气
func (r *SearchRepository) BatchUpdateVods(ids []int64, field string, value int64) error {
	f, ok := vodBatchColumns[field]
	if !ok {
		return fmt.Errorf("不支持批量设置 %q", field)
	}
	if value < f.min || value > f.max {
		return fmt.Errorf("取值需在 %d - %d 之间", f.min, f.max)
	}
	if len(ids) == 0 {
		return errors.New("请先选择视频")
	}
	if err := db.Mdb.Model(&Vod{}).Where("vod_id IN ?", ids).Update(f.column, value).Error; err != nil {
		return err
	}
	if field == "status" {
		// 审核状态影响前台可见性, 关键字搜索缓存需要重建
		db.Mdb.Exec(fmt.Sprintf("TRUNCATE TABLE %s", VodSearch{}.TableName()))
	}
	VodsChanged(ids, vodSchemes(ids))
	return nil
}

// IsPublished 视频是否已审核 (前台可见)
func (r *SearchRepository) IsPublished(vodId int64) bool {
	var count int64
	db.Mdb.Model(&Vod{}).Where("vod_id = ? AND vod_status = ?", vodId, vodStatusOn).Count(&count)
	return count > 0
}

// topTags 按分数从高到低取 ZSet 中排名 0..stop 的标签 (stop 为 -1 时取全部)
func topTags(key string, stop int64) []string {
	return db.Rdb.ZRangeArgs(db.Cxt, redis.ZRangeArgs{Key: key, Start: 0, Stop: stop, Rev: true}).Val()
}
