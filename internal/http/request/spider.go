package request

// SpiderStartRequest 开始 / 中止采集任务请求参数; Time 为采集最近 x 小时 (负数为全部)
type SpiderStartRequest struct {
	Id   string `json:"id"`
	Time int    `json:"time"`
}

// CollectClearRequest 清空采集接口的全部视频 (需要当前账户密码确认)
type CollectClearRequest struct {
	Id       string `json:"id"`
	Password string `json:"password"`
}
