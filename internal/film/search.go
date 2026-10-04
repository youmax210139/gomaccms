package film

// SearchInfo 影片检索信息 (vod 表的摘要视图): 后台影视信息列表、首页热门影片、检索标签等使用;
// ID 与 Mid 都是 vod_id, 保留 ID 字段供后台页面沿用
type SearchInfo struct {
	ID           uint    `json:"ID"`
	Mid          int64   `json:"mid"`
	Cid          int64   `json:"cid"`
	Pid          int64   `json:"pid"`
	Name         string  `json:"name"`
	SubTitle     string  `json:"subTitle"`
	CName        string  `json:"cName"`
	ClassTag     string  `json:"classTag"`
	Area         string  `json:"area"`
	Language     string  `json:"language"`
	Year         int64   `json:"year"`
	Initial      string  `json:"initial"`
	Score        float64 `json:"score"`
	UpdateStamp  int64   `json:"updateStamp"`
	Hits         int64   `json:"hits"`
	State        string  `json:"state"`
	Remarks      string  `json:"remarks"`
	ReleaseStamp int64   `json:"releaseStamp"`
	Status       int     `json:"status"` // 审核: 1 已审核 0 未审核
	Level        int     `json:"level"`  // 推荐 0-9
	Lock         int     `json:"lock"`   // 1 锁定
}

// Tag 影片分类标签结构体
type Tag struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}
