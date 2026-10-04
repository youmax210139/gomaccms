package request

// LanguageRequest 新增 (IsNew) / 修改语言, 或切换启用状态
type LanguageRequest struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	LibreCode string `json:"libreCode"`
	Enabled   bool   `json:"enabled"`
	Sort      int64  `json:"sort"`
	IsNew     bool   `json:"isNew"`
}
