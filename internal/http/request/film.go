package request

import "gomaccms/internal/film"

// FilmDetailRequest 手动添加影片请求参数
type FilmDetailRequest struct {
	Id           int64    `json:"id"`
	Cid          int64    `json:"cid"`
	Pid          int64    `json:"pid"`
	Name         string   `json:"name"`
	Picture      string   `json:"picture"`
	PlayFrom     []string `json:"playFrom"`
	DownFrom     string   `json:"DownFrom"`
	PlayLink     string   `json:"playLink"`
	DownloadLink string   `json:"downloadLink"`
	SubTitle     string   `json:"subTitle"`
	CName        string   `json:"cName"`
	EnName       string   `json:"enName"`
	Initial      string   `json:"initial"`
	ClassTag     string   `json:"classTag"`
	Actor        string   `json:"actor"`
	Director     string   `json:"director"`
	Writer       string   `json:"writer"`
	Remarks      string   `json:"remarks"`
	ReleaseDate  string   `json:"releaseDate"`
	Area         string   `json:"area"`
	Language     string   `json:"language"`
	Year         string   `json:"year"`
	State        string   `json:"state"`
	UpdateTime   string   `json:"updateTime"`
	AddTime      string   `json:"addTime"`
	DbId         int64    `json:"dbId"`
	DbScore      string   `json:"dbScore"`
	Hits         int64    `json:"hits"`
	Content      string   `json:"content"`
}

// FilmClassUpdateRequest 行内修改分类的状态或排序, 未传的字段 (nil) 保持不变
type FilmClassUpdateRequest struct {
	Id   int64  `json:"id"`
	Show *bool  `json:"show"`
	Sort *int64 `json:"sort"`
}

// FilmClassSaveRequest 新增 (Id 为 0) / 编辑分类
type FilmClassSaveRequest struct {
	Id   int64  `json:"id"`
	Type int    `json:"type"`
	Pid  int64  `json:"pid"`
	Name string `json:"name"`
	Slug string `json:"slug"`
	Show bool   `json:"show"`
	Sort int64  `json:"sort"`
	// I18n 各语言的名称与 SEO (留空的字段前台使用原文)
	I18n map[string]film.CategoryText `json:"i18n"`
	// SchemeId 新增分类所属的分类方案 (修改时忽略, 方案创建后不变)
	SchemeId int64 `json:"schemeId"`
}

// CategorySchemeRequest 新增 (Id 为 0) / 修改分类方案
type CategorySchemeRequest struct {
	Id          int64    `json:"id"`
	Name        string   `json:"name"`
	Sort        int64    `json:"sort"`
	DefaultLang string   `json:"defaultLang"`
	Langs       []string `json:"langs"`
}

// FilmClassBatchRequest 批量操作分类 (删除 / 修改状态 / 转移影片)
type FilmClassBatchRequest struct {
	Ids    []int64 `json:"ids"`
	Show   bool    `json:"show"`
	Target int64   `json:"target"`
}

// FilmBatchUpdateRequest 批量设置视频: Field 为 level (推荐 0-9) / status (审核 0/1) / lock (锁定 0/1) / hits (人气)
type FilmBatchUpdateRequest struct {
	Ids   []int64 `json:"ids"`
	Field string  `json:"field"`
	Value int64   `json:"value"`
}

// PlayerRequest 新增 (IsNew) / 修改播放器, 或切换启用状态
type PlayerRequest struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Status bool   `json:"status"`
	Sort   int64  `json:"sort"`
	Remark string `json:"remark"`
	IsNew  bool   `json:"isNew"`
}
