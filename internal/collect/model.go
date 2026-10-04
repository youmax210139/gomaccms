package collect

import (
	"net/url"
	"strings"
	"time"
)

type CollectResultModel int

const (
	JsonResult CollectResultModel = iota
	XmlResult
)

type ResourceType int

func (rt ResourceType) GetActionType() string {
	var ac string
	switch rt {
	case CollectVideo:
		ac = "detail"
	case CollectArticle:
		ac = "article"
	case CollectActor:
		ac = "actor"
	case CollectRole:
		ac = "role"
	case CollectWebSite:
		ac = "web"
	default:
		ac = "detail"
	}
	return ac
}

const (
	CollectVideo = iota
	CollectArticle
	CollectActor
	CollectRole
	CollectWebSite
)

// DataOperation 数据操作: 采集到的数据是新增、更新还是两者都做
type DataOperation int

const (
	OperationAll    DataOperation = iota // 新增+更新
	OperationAdd                         // 仅新增 (只建立不存在的数据)
	OperationUpdate                      // 仅更新 (只更新已存在的数据)
)

// FilterMode 地址过滤: 过滤代码/过滤年份作用于哪种数据操作
type FilterMode int

const (
	FilterNone   FilterMode = iota // 不过滤
	FilterAll                      // 新增+更新
	FilterAdd                      // 新增
	FilterUpdate                   // 更新
)

// SyncImageMode 同步图片设置
type SyncImageMode int

const (
	SyncImageGlobal SyncImageMode = iota // 跟随全局 (目前没有全局开关, 等同关闭)
	SyncImageOn                          // 开启
	SyncImageOff                         // 关闭
)

// FilmSource 采集接口 (collect_sources 表), 所有资源类型 (视频/文章/演员/角色/网站) 共用
type FilmSource struct {
	Id          string             `json:"id" gorm:"primaryKey"`
	Name        string             `json:"name"`
	Uri         string             `json:"uri"`
	Params      string             `json:"params"` // 附加参数, 一般 & 开头, 如 &ct=1
	ResultModel CollectResultModel `json:"resultModel"`
	CollectType ResourceType       `json:"collectType"`
	Operation   DataOperation      `json:"operation"`
	FilterMode  FilterMode         `json:"filterMode"`
	FilterCode  string             `json:"filterCode"` // 过滤代码, 逗号分隔, 如 youku,iqiyi
	FilterYear  string             `json:"filterYear"` // 过滤年份, 逗号分隔, 如 2022,2023
	SyncImage   SyncImageMode      `json:"syncImage"`
	State       bool               `json:"state"`
	Interval    int                `json:"interval"`
	CreatedAt   time.Time          `json:"-"`
	UpdatedAt   time.Time          `json:"-"`
}

// TableName 采集接口表表名
func (s FilmSource) TableName() string {
	return "collect_sources"
}

// SyncPictures 当前站点是否同步图片
func (s FilmSource) SyncPictures() bool {
	return s.SyncImage == SyncImageOn
}

// Query 以附加参数作为每次请求接口的基础参数
func (s FilmSource) Query() url.Values {
	v, _ := url.ParseQuery(strings.TrimLeft(s.Params, "&?"))
	if v == nil {
		v = url.Values{}
	}
	return v
}
