package cron

import "github.com/robfig/cron/v3"

// FilmCollectTask 影视采集任务
type FilmCollectTask struct {
	Id     string       `json:"id"`     // 唯一标识uid
	Ids    []string     `json:"ids"`    // 采集站id列表
	Cid    cron.EntryID `json:"cid"`    // 定时任务Id
	Time   int          `json:"time"`   // 采集时长, 最新x小时更新的内容
	Spec   string       `json:"spec"`   // 执行周期 cron表达式
	Model  int          `json:"model"`  // 任务类型, 0 - 自动更新已启用站点 || 1 - 更新Ids中的资源站数据 (2 采集重试 / 3 影片信息同步 已废弃)
	State  bool         `json:"state"`  // 状态 开启 | 禁用
	Remark string       `json:"remark"` // 任务备注信息
}
