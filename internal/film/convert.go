package film

import (
	"strings"
	"time"
)

/*
	系统内部对象想换转换
*/

// CovertFilmDetailVo 将 FilmDetailVo 转化为 MovieDetail
func CovertFilmDetailVo(fd FilmDetailVo) (MovieDetail, error) {
	t, err := time.ParseInLocation(time.DateTime, fd.AddTime, time.Local)
	md := MovieDetail{
		Mid:         fd.Id,
		Cid:         fd.Cid,
		Pid:         fd.Pid,
		Name:        fd.Name,
		Picture:     fd.Picture,
		SubTitle:    fd.SubTitle,
		CName:       fd.CName,
		EnName:      fd.EnName,
		Initial:     fd.Initial,
		ClassTag:    fd.ClassTag,
		Actor:       fd.Actor,
		Director:    fd.Director,
		Writer:      fd.Writer,
		Blurb:       fd.Content,
		Remarks:     fd.Remarks,
		ReleaseDate: fd.ReleaseDate,
		Area:        fd.Area,
		Language:    fd.Language,
		Year:        fd.Year,
		State:       fd.State,
		UpdateTime:  fd.UpdateTime,
		AddTime:     t.Unix(),
		DbId:        fd.DbId,
		DbScore:     fd.DbScore,
		Hits:        fd.Hits,
		Content:     fd.Content,
		PlayFrom:    fd.PlayFrom,
		DownFrom:    fd.DownFrom,
	}
	// 播放源以 $$$ 分隔, 只保留 m3u8 / mp4 地址
	md.PlayList = GenFilmPlayList(fd.PlayLink, "$$$")
	md.DownloadList = GenFilmPlayList(fd.DownloadLink, "$$$")
	return md, err
}

func ConvertCategoryList(tree CategoryTree) []Category {
	var cl = []Category{Category{Id: tree.Id, Pid: tree.Pid, Name: tree.Name, Show: tree.Show}}
	for _, c := range tree.Children {
		cl = append(cl, Category{Id: c.Id, Pid: c.Pid, Name: c.Name, Show: c.Show})
		if len(c.Children) > 0 {
			for _, subC := range c.Children {
				cl = append(cl, Category{Id: subC.Id, Pid: subC.Pid, Name: subC.Name, Show: subC.Show})
			}
		}
	}
	return cl
}

func GenFilmPlayList(playUrl, separator string) MoviePlayList {
	var res MoviePlayList
	if separator != "" {
		// 1. 通过分隔符切分播放源地址
		for _, l := range strings.Split(playUrl, separator) {
			// 2.只对m3u8播放源 和 .mp4下载地址进行处理
			if strings.Contains(l, ".m3u8") || strings.Contains(l, ".mp4") {
				// 2. 将每组播放源对应的播放列表信息存储到列表中
				res = append(res, ConvertPlayUrl(l))
			}
		}
	} else {
		// 1.只对m3u8播放源 和 .mp4下载地址进行处理
		if strings.Contains(playUrl, ".m3u8") || strings.Contains(playUrl, ".mp4") {
			// 2. 将每组播放源对应的播放列表信息存储到列表中
			res = append(res, ConvertPlayUrl(playUrl))
		}
	}
	return res
}

func GenAllFilmPlayList(playUrl, separator string) MoviePlayList {
	var res MoviePlayList
	if separator != "" {
		// 1. 通过分隔符切分播放源地址
		for _, l := range strings.Split(playUrl, separator) {
			// 将playUrl中的所有播放格式链接均进行转换保存
			res = append(res, ConvertPlayUrl(l))
		}
		return res
	}
	// 将playUrl中的所有播放格式链接均进行转换保存
	res = append(res, ConvertPlayUrl(playUrl))
	return res
}

func ConvertPlayUrl(playUrl string) []PlayItem {
	// 对每个片源的集数和播放地址进行分割 Episode$Link#Episode$Link
	var l []PlayItem
	for _, p := range strings.Split(playUrl, "#") {
		// 处理 Episode$Link 形式的播放信息
		if strings.Contains(p, "$") {
			l = append(l, PlayItem{
				Episode: strings.Split(p, "$")[0],
				Link:    strings.Split(p, "$")[1],
			})
		} else {
			l = append(l, PlayItem{
				Episode: "(｀・ω・´)",
				Link:    p,
			})
		}
	}
	return l
}
