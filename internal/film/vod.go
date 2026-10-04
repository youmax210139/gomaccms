package film

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Vod 影片表 vod (栏位命名参照苹果 CMS 的 mac_vod), 取代原先的 search 与 movie_details 两张表.
// 仓库对外仍以 MovieDetail / SearchInfo / MovieBasicInfo 交换数据 (模板、首页、后台沿用原字段名),
// 只在读写数据库时与 Vod 互相转换.
type Vod struct {
	VodId          int64   `gorm:"column:vod_id;primaryKey"` // 本站影片ID (自增)
	TypeId         int64   `gorm:"column:type_id"`           // 分类ID (cid)
	TypeId1        int64   `gorm:"column:type_id_1"`         // 一级分类ID (pid)
	TypeName       string  `gorm:"column:type_name"`
	VodName        string  `gorm:"column:vod_name"`
	VodNameKey     string  `gorm:"column:vod_name_key"` // 比对用片名 (去空白与标点, 小写), 见 NameKey
	VodSub         string  `gorm:"column:vod_sub"`
	VodEn          string  `gorm:"column:vod_en"`
	VodStatus      int     `gorm:"column:vod_status"` // 审核: 1 已审核 0 未审核 (前台不显示)
	VodLevel       int     `gorm:"column:vod_level"`  // 推荐 0-9, 0 为未推荐
	VodLock        int     `gorm:"column:vod_lock"`   // 1 锁定 (采集不再更新)
	VodLetter      string  `gorm:"column:vod_letter"`
	VodClass       string  `gorm:"column:vod_class"` // 剧情标签
	VodPic         string  `gorm:"column:vod_pic"`
	VodActor       string  `gorm:"column:vod_actor"`
	VodDirector    string  `gorm:"column:vod_director"`
	VodWriter      string  `gorm:"column:vod_writer"`
	VodRemarks     string  `gorm:"column:vod_remarks"`
	VodPubdate     string  `gorm:"column:vod_pubdate"`
	VodArea        string  `gorm:"column:vod_area"`
	VodLang        string  `gorm:"column:vod_lang"`
	VodYear        string  `gorm:"column:vod_year"`
	VodState       string  `gorm:"column:vod_state"`
	VodHits        int64   `gorm:"column:vod_hits"`
	VodScore       float64 `gorm:"column:vod_score"`
	VodTime        int64   `gorm:"column:vod_time"`     // 更新时间 (Unix)
	VodTimeAdd     int64   `gorm:"column:vod_time_add"` // 添加时间 (Unix)
	VodDoubanId    int64   `gorm:"column:vod_douban_id"`
	VodDoubanScore float64 `gorm:"column:vod_douban_score"`
	VodContent     string  `gorm:"column:vod_content"`
	VodPlayFrom    string  `gorm:"column:vod_play_from"` // 播放组名称, $$$ 分隔
	VodPlayUrl     string  `gorm:"column:vod_play_url"`  // 播放地址, 组间 $$$, 组内 集名$地址#集名$地址
	VodDownFrom    string  `gorm:"column:vod_down_from"`
	VodDownUrl     string  `gorm:"column:vod_down_url"`
}

// TableName 影片表表名
func (Vod) TableName() string {
	return "vod"
}

// VodType 影片所属的分类 (vod_type 表): 一部影片可以属于多个分类 (可跨分类方案), 列表、首页、筛选等都按这里查询;
// Status 为 0 表示该分类被停用
type VodType struct {
	VodId    int64 `gorm:"column:vod_id;primaryKey;autoIncrement:false"`
	TypeId   int64 `gorm:"column:type_id;primaryKey;autoIncrement:false"`
	TypeId1  int64 `gorm:"column:type_id_1"`
	SchemeId int64 `gorm:"column:scheme_id"`
	Status   int   `gorm:"column:status"`
}

// TableName 影片分类关联表表名
func (VodType) TableName() string {
	return "vod_type"
}

// VodSearch 关键字搜索结果缓存 (参照 mac_vod_search)
type VodSearch struct {
	SearchKey         string `gorm:"column:search_key;primaryKey"`
	SearchWord        string `gorm:"column:search_word"`
	SearchField       string `gorm:"column:search_field"`
	SearchHitCount    int64  `gorm:"column:search_hit_count"`
	SearchLastHitTime int64  `gorm:"column:search_last_hit_time"`
	SearchUpdateTime  int64  `gorm:"column:search_update_time"`
	SearchResultCount int    `gorm:"column:search_result_count"`
	SearchResultIds   string `gorm:"column:search_result_ids"`
}

// TableName 搜索缓存表表名
func (VodSearch) TableName() string {
	return "vod_search"
}

const (
	playGroupSep = "$$$" // 播放组分隔符
	vodStatusOn  = 1
	vodStatusOff = 0
)

// vodBasicColumns 列表类查询只取的栏位 (不含详情与播放地址)
var vodBasicColumns = []string{"vod_id", "type_id", "type_id_1", "type_name", "vod_name", "vod_sub", "vod_letter",
	"vod_status", "vod_level", "vod_lock",
	"vod_class", "vod_pic", "vod_actor", "vod_director", "vod_remarks", "vod_area", "vod_lang", "vod_year",
	"vod_state", "vod_hits", "vod_score", "vod_time", "vod_time_add"}

// vodUpdateColumns 已存在的影片再次采集时更新的栏位 (名称、分类等可能被后台修改过, 不覆盖)
var vodUpdateColumns = []string{"vod_play_from", "vod_play_url", "vod_down_from", "vod_down_url", "vod_remarks",
	"vod_state", "vod_time", "vod_time_add", "vod_score", "vod_douban_score", "vod_hits"}

var yearPattern = regexp.MustCompile(`[12][0-9]{3}`)

// NameKey 入库重复规则比对用的片名: 去掉空白与标点并转小写 (与 migration 00014 的 SQL 规则一致)
func NameKey(name string) string {
	key := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || unicode.IsPunct(r) {
			return -1
		}
		return unicode.ToLower(r)
	}, name)
	return clip(key, 255)
}

// clip 按字符数截断, 避免超出 varchar 长度导致整条写入失败
func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}

func parseScore(s string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if f < 0 || f > 99.9 {
		return 0
	}
	return f
}

// EncodePlayList 播放列表转为苹果 CMS 的播放地址格式
func EncodePlayList(pl MoviePlayList) string {
	groups := make([]string, 0, len(pl))
	for _, g := range pl {
		items := make([]string, 0, len(g))
		for _, p := range g {
			items = append(items, p.Episode+"$"+p.Link)
		}
		groups = append(groups, strings.Join(items, "#"))
	}
	return strings.Join(groups, playGroupSep)
}

func decodePlayList(s string) MoviePlayList {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return GenAllFilmPlayList(s, playGroupSep)
}

// NewVod 将影片详情转为 vod 表记录
func NewVod(m MovieDetail) Vod {
	updated := time.Now().Unix()
	if t, err := time.ParseInLocation(time.DateTime, m.UpdateTime, time.Local); err == nil && t.Unix() > 0 {
		updated = t.Unix()
	}
	year := yearPattern.FindString(m.Year)
	if year == "" {
		year = yearPattern.FindString(m.ReleaseDate)
	}
	score := parseScore(m.DbScore)
	return Vod{
		VodId: m.Mid, TypeId: m.Cid, TypeId1: m.Pid, TypeName: clip(m.CName, 60),
		VodName: clip(m.Name, 255), VodNameKey: NameKey(m.Name), VodSub: clip(m.SubTitle, 255), VodEn: clip(m.EnName, 255), VodStatus: vodStatusOn,
		VodLetter: clip(m.Initial, 10), VodClass: clip(m.ClassTag, 255), VodPic: clip(m.Picture, 1024),
		VodActor: clip(m.Actor, 255), VodDirector: clip(m.Director, 255), VodWriter: clip(m.Writer, 255),
		VodRemarks: clip(m.Remarks, 255), VodPubdate: clip(m.ReleaseDate, 100), VodArea: clip(m.Area, 255),
		VodLang: clip(m.Language, 255), VodYear: year, VodState: clip(m.State, 255), VodHits: max(m.Hits, 0),
		VodScore: score, VodTime: updated, VodTimeAdd: max(m.AddTime, 0), VodDoubanId: max(m.DbId, 0),
		VodDoubanScore: score, VodContent: m.Content,
		VodPlayFrom: strings.Join(m.PlayFrom, playGroupSep), VodPlayUrl: EncodePlayList(m.PlayList),
		VodDownFrom: clip(m.DownFrom, 255), VodDownUrl: EncodePlayList(m.DownloadList),
	}
}

// Detail vod 记录转为影片详情
func (v Vod) Detail() MovieDetail {
	var from FromList
	if v.VodPlayFrom != "" {
		from = strings.Split(v.VodPlayFrom, playGroupSep)
	}
	return MovieDetail{
		Id: v.VodId, Mid: v.VodId, Cid: v.TypeId, Pid: v.TypeId1, Name: v.VodName, Picture: v.VodPic,
		SubTitle: v.VodSub, CName: v.TypeName, EnName: v.VodEn, Initial: v.VodLetter, ClassTag: v.VodClass,
		Actor: v.VodActor, Director: v.VodDirector, Writer: v.VodWriter, Remarks: v.VodRemarks,
		ReleaseDate: v.VodPubdate, Area: v.VodArea, Language: v.VodLang, Year: v.VodYear, State: v.VodState,
		UpdateTime: time.Unix(v.VodTime, 0).Format(time.DateTime), AddTime: v.VodTimeAdd, DbId: v.VodDoubanId,
		DbScore: fmt.Sprintf("%.1f", v.VodDoubanScore), Hits: v.VodHits, Content: v.VodContent,
		PlayFrom: from, DownFrom: v.VodDownFrom, PlayList: decodePlayList(v.VodPlayUrl),
		DownloadList: decodePlayList(v.VodDownUrl),
	}
}

// Search vod 记录转为检索信息 (后台影视信息列表、首页热门等)
func (v Vod) Search() SearchInfo {
	year, _ := strconv.ParseInt(v.VodYear, 10, 64)
	return SearchInfo{
		ID: uint(v.VodId), Mid: v.VodId, Cid: v.TypeId, Pid: v.TypeId1, Name: v.VodName, SubTitle: v.VodSub,
		CName: v.TypeName, ClassTag: v.VodClass, Area: v.VodArea, Language: v.VodLang, Year: year,
		Initial: v.VodLetter, Score: v.VodScore, UpdateStamp: v.VodTime, Hits: v.VodHits, State: v.VodState,
		Remarks: v.VodRemarks, ReleaseStamp: v.VodTimeAdd, Status: v.VodStatus, Level: v.VodLevel, Lock: v.VodLock,
	}
}

// Basic vod 记录转为影片基本信息 (列表卡片)
func (v Vod) Basic() MovieBasicInfo {
	return MovieBasicInfo{Id: v.VodId, Cid: v.TypeId, Pid: v.TypeId1, Name: v.VodName, SubTitle: v.VodSub,
		CName: v.TypeName, State: v.VodState, Picture: v.VodPic, Actor: v.VodActor, Director: v.VodDirector,
		Blurb: v.VodContent, Remarks: v.VodRemarks, Area: v.VodArea, Year: v.VodYear}
}
