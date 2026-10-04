package spider

import (
	"fmt"
	"gomaccms/internal/collect"
	"gomaccms/internal/film"
	"gomaccms/internal/util"
	"strings"
)

/*
	处理 不同结构体数据之间的转化
	统一转化为内部结构体
*/

// ConvertCategoryList 将分类树形数据转化为list类型
// ConvertFilmDetails 批量处理影片详情信息
func ConvertFilmDetails(details []collect.FilmDetail) []film.MovieDetail {
	var dl []film.MovieDetail
	for _, d := range details {
		dl = append(dl, ConvertFilmDetail(d))
	}
	return dl

}

// ConvertFilmDetail 将影片详情数据处理转化为 film.MovieDetail
func ConvertFilmDetail(detail collect.FilmDetail) film.MovieDetail {
	/*
		对需数据进行相应的简化处理
		1.对常见分割符进行统一化处理
		2.如果演员和导演名单过长,则进行截断, 最多只保留3个
	*/

	md := film.MovieDetail{
		Mid:      detail.VodID,
		Cid:      detail.TypeID,
		Pid:      detail.TypeID1,
		Name:     detail.VodName,
		Picture:  detail.VodPic,
		DownFrom: detail.VodDownFrom,
		SubTitle: detail.VodSub,
		CName:    detail.TypeName,
		EnName:   detail.VodEn,
		Initial:  detail.VodLetter,
		ClassTag: detail.VodClass,
		Actor:    util.TruncateBySep(detail.VodActor, 3),
		Director: util.TruncateBySep(detail.VodDirector, 2),
		Writer:   util.TruncateBySep(detail.VodWriter, 2),
		//Blurb:       detail.VodBlurb,
		Blurb:       "", // blurb 和 content 内容重复度过高, 且内存占用过高, 所以舍弃简介字段
		Remarks:     detail.VodRemarks,
		ReleaseDate: detail.VodPubDate,
		Area:        detail.VodArea,
		Language:    detail.VodLang,
		Year:        detail.VodYear,
		State:       detail.VodState,
		UpdateTime:  detail.VodTime,
		AddTime:     detail.VodTimeAdd,
		DbId:        detail.VodDouBanID,
		DbScore:     detail.VodDouBanScore,
		Hits:        detail.VodHits,
		Content:     detail.VodContent,
	}
	// 播放组以 vod_play_note (一般为 $$$) 分隔, 播放组代码与播放地址一一对应; 只保留 m3u8 / mp4 播放组
	sep := detail.VodPlayNote
	if sep == "" {
		sep = "$$$"
	}
	froms := strings.Split(detail.VodPlayFrom, sep)
	for i, u := range strings.Split(detail.VodPlayURL, sep) {
		if !strings.Contains(u, ".m3u8") && !strings.Contains(u, ".mp4") {
			continue
		}
		code := fmt.Sprintf("play%d", i+1)
		if i < len(froms) && strings.TrimSpace(froms[i]) != "" {
			code = strings.TrimSpace(froms[i])
		}
		md.PlayFrom = append(md.PlayFrom, code)
		md.PlayList = append(md.PlayList, film.ConvertPlayUrl(u))
	}
	return md
}

// ----------------------------------Provide API---------------------------------------------------
