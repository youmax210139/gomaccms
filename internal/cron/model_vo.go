package cron

// FilmCronVo 影视更新任务请求参数
type FilmCronVo struct {
	Ids    []string `json:"ids"`    // 定时任务关联的资源站Id
	Time   int      `json:"time"`   // 更新最近几小时内更新的影片
	Spec   string   `json:"spec"`   // cron表达式
	Model  int      `json:"model"`  // 任务类型, 0 - 自动更新已启用站点 || 1 - 更新Ids中的资源站数据
	State  bool     `json:"state"`  // 任务状态 开启 | 关闭
	Remark string   `json:"remark"` // 备注信息
}

// CronTaskVo 定时任务数据response
type CronTaskVo struct {
	FilmCollectTask
	PreV string `json:"preV"` // 上次执行时间
	Next string `json:"next"` // 下次执行时间
}
