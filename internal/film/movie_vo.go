package film

import "gomaccms/internal/paging"

// SearchTagsVO 搜索标签请求参数
type SearchTagsVO struct {
	Pid      int64  `json:"pid"`
	Cid      int64  `json:"cid"`
	Plot     string `json:"plot"`
	Area     string `json:"area"`
	Language string `json:"language"`
	Year     int64  `json:"year"`
	Sort     string `json:"sort"`
}

// SearchVo 影片信息搜索参数
type SearchVo struct {
	Name      string       `json:"name"`
	Pid       int64        `json:"pid"`
	Cid       int64        `json:"cid"`
	Plot      string       `json:"plot"`
	Area      string       `json:"area"`
	Language  string       `json:"language"`
	Year      int64        `json:"year"`
	Remarks   string       `json:"remarks"`
	SchemeId  int64        `json:"schemeId"` // 分类方案 (未选分类时按方案筛选)
	Player    string       `json:"player"`   // 播放器 (播放组代码)
	Picture   string       `json:"picture"`  // 图片: has 有图片 / none 无图片
	Sort      string       `json:"sort"`     // 排序: time 更新时间 / add 添加时间 / hits 人气 / score 评分 / id 编号
	Status    string       `json:"status"`   // 审核: "" 全部 / 1 已审核 / 0 未审核
	Level     string       `json:"level"`    // 推荐: "" 全部 / 0-9
	Lock      string       `json:"lock"`     // 锁定: "" 全部 / 1 已锁定 / 0 未锁定
	BeginTime int64        `json:"beginTime"`
	EndTime   int64        `json:"endTime"`
	Paging    *paging.Page `json:"paging"`
}

// FilmDetailVo 添加影片对象
type FilmDetailVo struct {
	Id           int64    `json:"id"`
	Cid          int64    `json:"cid"`
	Pid          int64    `json:"pid"`
	Name         string   `json:"name"`
	Picture      string   `json:"picture"`
	PlayFrom     []string `json:"playFrom"`
	DownFrom     string   `json:"DownFrom"`
	PlayLink     string   `json:"playLink"`
	DownloadLink string   `json:"downloadLink"`
	SubTitle     string   `json:"subTitle"`
	CName        string   `json:"cName"`
	EnName       string   `json:"enName"`
	Initial      string   `json:"initial"`
	ClassTag     string   `json:"classTag"`
	Actor        string   `json:"actor"`
	Director     string   `json:"director"`
	Writer       string   `json:"writer"`
	Remarks      string   `json:"remarks"`
	ReleaseDate  string   `json:"releaseDate"`
	Area         string   `json:"area"`
	Language     string   `json:"language"`
	Year         string   `json:"year"`
	State        string   `json:"state"`
	UpdateTime   string   `json:"updateTime"`
	AddTime      string   `json:"addTime"`
	DbId         int64    `json:"dbId"`
	DbScore      string   `json:"dbScore"`
	Hits         int64    `json:"hits"`
	Content      string   `json:"content"`
}

// PlayLinkVo 多站点播放链接数据列表
type PlayLinkVo struct {
	Id       string     `json:"id"`
	Name     string     `json:"name"`
	LinkList []PlayItem `json:"linkList"`
}

// MovieDetailVo 影片详情数据, 播放源合并版
type MovieDetailVo struct {
	Id           int64         `json:"id" gorm:"primaryKey"`
	Mid          int64         `json:"mid"`
	Cid          int64         `json:"cid"`
	Pid          int64         `json:"pid"`
	Name         string        `json:"name"`
	Picture      string        `json:"picture"`
	SubTitle     string        `json:"subTitle"`
	CName        string        `json:"cName"`
	EnName       string        `json:"enName"`
	Initial      string        `json:"initial"`
	ClassTag     string        `json:"classTag"`
	Actor        string        `json:"actor"`
	Director     string        `json:"director"`
	Writer       string        `json:"writer"`
	Blurb        string        `json:"blurb"`
	Remarks      string        `json:"remarks"`
	ReleaseDate  string        `json:"releaseDate"`
	Area         string        `json:"area"`
	Language     string        `json:"language"`
	Year         string        `json:"year"`
	State        string        `json:"state"`
	UpdateTime   string        `json:"updateTime"`
	AddTime      int64         `json:"addTime"`
	DbId         int64         `json:"dbId"`
	DbScore      string        `json:"dbScore"`
	Hits         int64         `json:"hits"`
	Content      string        `json:"content"`
	PlayFrom     FromList      `json:"playFrom" gorm:"type:json"`
	DownFrom     string        `json:"DownFrom"`
	List         []PlayLinkVo  `json:"list"`
	DownloadList MoviePlayList `json:"downloadList" gorm:"type:json"`
}

// ConvertMovieDetailVo 整合详情信息
func ConvertMovieDetailVo(d MovieDetail, l []PlayLinkVo) MovieDetailVo {
	return MovieDetailVo{
		Id: d.Id, Mid: d.Mid, Cid: d.Cid, Pid: d.Pid, Name: d.Name, Picture: d.Picture,
		SubTitle: d.SubTitle, CName: d.CName, EnName: d.EnName, Initial: d.Initial,
		ClassTag: d.ClassTag, Actor: d.Actor, Director: d.Director, Writer: d.Writer,
		Blurb: "", Remarks: d.Remarks, ReleaseDate: d.ReleaseDate, Area: d.Area,
		Language: d.Language, Year: d.Year, State: d.State, UpdateTime: d.UpdateTime,
		AddTime: d.AddTime, DbId: d.DbId, DbScore: d.DbScore, Hits: d.Hits, Content: d.Content,
		PlayFrom: d.PlayFrom, DownFrom: d.DownFrom, List: l,
	}
}
