package request

// FilmCronRequest 定时任务新增请求参数
type FilmCronRequest struct {
	Ids    []string `json:"ids"`
	Time   int      `json:"time"`
	Spec   string   `json:"spec"`
	Model  int      `json:"model"`
	State  bool     `json:"state"`
	Remark string   `json:"remark"`
}

// FilmCronTaskRequest 定时任务更新/状态变更请求参数
type FilmCronTaskRequest struct {
	Id     string   `json:"id"`
	Ids    []string `json:"ids"`
	Time   int      `json:"time"`
	State  bool     `json:"state"`
	Remark string   `json:"remark"`
}
