package request

// FilmSourceRequest 采集源新增/更新/测试请求参数
type FilmSourceRequest struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	Uri         string `json:"uri"`
	Params      string `json:"params"`
	ResultModel int    `json:"resultModel"`
	Grade       int    `json:"grade"`
	CollectType int    `json:"collectType"`
	Operation   int    `json:"operation"`
	FilterMode  int    `json:"filterMode"`
	FilterCode  string `json:"filterCode"`
	FilterYear  string `json:"filterYear"`
	SyncImage   int    `json:"syncImage"`
	State       bool   `json:"state"`
	Interval    int    `json:"interval"`
}

// FilmSourceStateRequest 采集接口启用状态切换请求参数
type FilmSourceStateRequest struct {
	Id    string `json:"id"`
	State bool   `json:"state"`
}

// CollectBindRequest 采集站分类绑定请求参数, CategoryIds 为空表示解除绑定
type CollectBindRequest struct {
	SourceId    string  `json:"sourceId"`
	TypeId      int64   `json:"typeId"`
	TypeName    string  `json:"typeName"`
	CategoryIds []int64 `json:"categoryIds"`
}

// CollectFilmIdsRequest 采集采集站中指定影片的请求参数
type CollectFilmIdsRequest struct {
	Id  string  `json:"id"`
	Ids []int64 `json:"ids"`
}
