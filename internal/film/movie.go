package film

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// Movie 影片基本信息
type Movie struct {
	Id       int64  `json:"id"`
	Name     string `json:"name"`
	Cid      int64  `json:"cid"`
	CName    string `json:"CName"`
	EnName   string `json:"enName"`
	Time     string `json:"time"`
	Remarks  string `json:"remarks"`
	PlayFrom string `json:"playFrom"`
}

// MovieBasicInfo 影片基本信息
type MovieBasicInfo struct {
	Id       int64  `json:"id"`
	Cid      int64  `json:"cid"`
	Pid      int64  `json:"pid"`
	Name     string `json:"name"`
	SubTitle string `json:"subTitle"`
	CName    string `json:"cName"`
	State    string `json:"state"`
	Picture  string `json:"picture"`
	Actor    string `json:"actor"`
	Director string `json:"director"`
	Blurb    string `json:"blurb"`
	Remarks  string `json:"remarks"`
	Area     string `json:"area"`
	Year     string `json:"year"`
}

// PlayItem 影视资源url信息
type PlayItem struct {
	Episode string `json:"episode"`
	Link    string `json:"link"`
}

// MoviePlayList 播放列表信息, 二维切片
type MoviePlayList [][]PlayItem

// FromList 播放来源切片
type FromList []string

// MovieDetail 影片详情信息
type MovieDetail struct {
	Id           int64         `json:"id" gorm:"primaryKey"`
	Mid          int64         `json:"mid"`
	Cid          int64         `json:"cid"`
	Pid          int64         `json:"pid"`
	Name         string        `json:"name"`
	Picture      string        `json:"picture"`
	SubTitle     string        `json:"subTitle"`
	CName        string        `json:"cName"`
	EnName       string        `json:"enName"`
	Initial      string        `json:"initial"`
	ClassTag     string        `json:"classTag"`
	Actor        string        `json:"actor"`
	Director     string        `json:"director"`
	Writer       string        `json:"writer"`
	Blurb        string        `json:"blurb" gorm:"type:text"`
	Remarks      string        `json:"remarks"`
	ReleaseDate  string        `json:"releaseDate"`
	Area         string        `json:"area"`
	Language     string        `json:"language"`
	Year         string        `json:"year"`
	State        string        `json:"state"`
	UpdateTime   string        `json:"updateTime"`
	AddTime      int64         `json:"addTime"`
	DbId         int64         `json:"dbId"`
	DbScore      string        `json:"dbScore"`
	Hits         int64         `json:"hits"`
	Content      string        `json:"content" gorm:"type:text"`
	PlayFrom     FromList      `json:"playFrom" gorm:"type:json"`
	DownFrom     string        `json:"DownFrom"`
	PlayList     MoviePlayList `json:"playList" gorm:"type:json"`
	DownloadList MoviePlayList `json:"downloadList" gorm:"type:json"`
	// Types 影片所属的本站分类ID (采集绑定可对应多个分类, 可跨分类方案); 为空时只属于 Cid
	Types []int64 `json:"types,omitempty" gorm:"-"`
}

// =================================== column序列化 接口========================================================

func (m *MoviePlayList) Scan(value interface{}) error {
	if value == nil {
		*m = nil
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return errors.New("MoviePlayList serialization failed, value is not []byte")
	}
	return json.Unmarshal(b, m)
}

func (m MoviePlayList) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	return json.Marshal(m)
}

func (fl *FromList) Scan(value interface{}) error {
	if value == nil {
		*fl = nil
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return errors.New("FromList serialization failed, value is not []byte")
	}
	return json.Unmarshal(b, fl)
}

func (fl FromList) Value() (driver.Value, error) {
	if fl == nil {
		return nil, nil
	}
	return json.Marshal(fl)
}

// ConvertBasicInfo 将Detail信息转化为basic信息
func ConvertBasicInfo(m MovieDetail) MovieBasicInfo {
	return MovieBasicInfo{Id: m.Mid, Cid: m.Cid, Pid: m.Pid, Name: m.Name, SubTitle: m.SubTitle,
		CName: m.CName, State: m.State, Picture: m.Picture, Actor: m.Actor, Director: m.Director, Blurb: m.Content,
		Remarks: m.Remarks, Area: m.Area, Year: m.Year}
}
