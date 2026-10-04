package request

// IdsRequest 批量操作的 ID 列表
type IdsRequest[T any] struct {
	Ids []T `json:"ids"`
}

// StatusRequest 启用 / 停用: 目标 ID 与新的状态值
type StatusRequest[ID, S any] struct {
	Id     ID `json:"id"`
	Status S  `json:"status"`
}
