package response

const (
	SUCCESS = 0
	FAILED  = -1
)

// Response http返回数据结构体
type Response struct {
	Code int    `json:"code"`
	Data any    `json:"data"`
	Msg  string `json:"msg"`
}
