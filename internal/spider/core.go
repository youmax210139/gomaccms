package spider

import (
	"encoding/json"
	"errors"
	"gomaccms/internal/collect"
	"gomaccms/internal/film"
	"gomaccms/internal/util"
	"log"
)

/*
	Spider 数据 爬取 & 处理 & 转换
*/

// ------------------------------------------------- JSON Collect -------------------------------------------------

// JsonCollect 处理返回值为JSON格式的采集数据
type JsonCollect struct {
}

// GetPageInfo 获取分页页数与数据总数 (默认 ac = detail)
func (jc *JsonCollect) GetPageInfo(r util.RequestInfo) (pageCount, total int, err error) {
	if len(r.Params.Get("ac")) <= 0 {
		r.Params.Set("ac", "detail")
	}
	r.Params.Set("pg", "1")
	util.ApiGet(&r)
	if len(r.Resp) <= 0 {
		return 0, 0, errors.New("response is empty")
	}
	res := collect.CommonPage{}
	if err = json.Unmarshal(r.Resp, &res); err != nil {
		return 0, 0, err
	}
	return res.PageCount, res.Total, nil
}

// GetClassList 获取采集站的分类列表 (ac=list 第一页返回的 class)
func (jc *JsonCollect) GetClassList(r util.RequestInfo) ([]collect.FilmClass, error) {
	r.Params.Set("ac", "list")
	r.Params.Set("pg", "1")
	r.Params.Del("t")
	r.Params.Del("h")
	util.ApiGet(&r)
	if len(r.Resp) <= 0 {
		return nil, errors.New("response is empty")
	}
	var page collect.FilmListPage
	if err := json.Unmarshal(r.Resp, &page); err != nil {
		return nil, err
	}
	return page.Class, nil
}

// GetFilmDetail 通过 RequestInfo 获取并解析出对应的 MovieDetail list
func (jc *JsonCollect) GetFilmDetail(r util.RequestInfo) (list []film.MovieDetail, err error) {
	// 防止json解析异常引发panic
	defer func() {
		if e := recover(); e != nil {
			log.Println("GetMovieDetail Failed : ", e)
		}
	}()
	// 设置分页请求参数
	r.Params.Set(`ac`, `detail`)
	util.ApiGet(&r)
	// 影视详情信息
	var detailPage collect.FilmDetailLPage
	//details := system.DetailListInfo{}
	// 如果返回数据为空则直接结束本次循环
	if len(r.Resp) <= 0 {
		err = errors.New(r.Err)
		return
	}
	// 序列化详情数据
	if err = json.Unmarshal(r.Resp, &detailPage); err != nil {
		return
	}
	// 将影视原始详情信息保存到redis中
	// 获取主站点uri
	//mc := system.GetCollectSourceListByGrade(system.MasterCollect)[0]
	//if mc.Uri == r.Uri {
	//	collect.BatchSaveOriginalDetail(detailPage.List)
	//}

	// 处理details信息
	list = ConvertFilmDetails(detailPage.List)
	return
}

// ------------------------------------------------- XML Collect -------------------------------------------------
