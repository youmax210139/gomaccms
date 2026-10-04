package film

import (
	"strings"
	"testing"
)

func sampleVod() Vod {
	return Vod{VodId: 9, TypeId: 6, TypeId1: 1, TypeName: "动作片", VodName: "流浪地球", VodNameKey: "流浪地球",
		VodSub: "The Wandering Earth", VodLetter: "L", VodClass: "科幻,冒险", VodPic: "https://img/x.jpg",
		VodActor: "吴京", VodDirector: "郭帆", VodWriter: "刘慈欣", VodRemarks: "HD", VodPubdate: "2019-02-05",
		VodArea: "大陆", VodLang: "国语", VodYear: "2019", VodState: "正片", VodContent: "<p>简介</p>",
		VodStatus: 1, VodLevel: 3, VodLock: 0, VodHits: 120, VodScore: 7.9, VodTime: 1700000000,
		VodPlayFrom: "ffm3u8$$$lzm3u8",
		VodPlayUrl:  "第01集$https://a/1.m3u8#第02集$https://a/2.m3u8$$$正片$https://b/1.m3u8"}
}

func TestVodEditRoundTrip(t *testing.T) {
	v := sampleVod()
	e := vodEditOf(v)
	if len(e.PlayGroups) != 2 || e.PlayGroups[0].From != "ffm3u8" || e.PlayGroups[0].Text != "第01集$https://a/1.m3u8\n第02集$https://a/2.m3u8" {
		t.Fatalf("play groups = %+v", e.PlayGroups)
	}
	got := v
	if err := e.applyTo(&got); err != nil {
		t.Fatal(err)
	}
	if got != v {
		t.Fatalf("round trip changed the row:\n got %+v\nwant %+v", got, v)
	}
}

func TestVodEditApply(t *testing.T) {
	v := sampleVod()
	e := vodEditOf(v)
	e.Name, e.Year, e.Lock = " 流浪地球 2 ", "2023", 1
	e.PlayGroups = []VodPlayGroup{{From: "lzm3u8", Text: "https://b/1.m3u8\n\n 预告 $ https://b/2.m3u8 "}, {From: "x", Text: "  "}}
	if err := e.applyTo(&v); err != nil {
		t.Fatal(err)
	}
	if v.VodName != "流浪地球 2" || v.VodNameKey != "流浪地球2" || v.VodYear != "2023" || v.VodLock != 1 {
		t.Fatalf("fields = %+v", v)
	}
	if v.VodPlayFrom != "lzm3u8" || v.VodPlayUrl != "https://b/1.m3u8#预告$https://b/2.m3u8" {
		t.Fatalf("play = %q / %q", v.VodPlayFrom, v.VodPlayUrl)
	}
	if v.VodTime != 1700000000 {
		t.Fatal("editing must not change vod_time")
	}
}

func TestVodEditValidate(t *testing.T) {
	cases := map[string]func(e *VodEdit){
		"empty name":     func(e *VodEdit) { e.Name = " " },
		"bad year":       func(e *VodEdit) { e.Year = "19" },
		"level > 9":      func(e *VodEdit) { e.Level = 10 },
		"score > 10":     func(e *VodEdit) { e.Score = 11 },
		"negative hits":  func(e *VodEdit) { e.Hits = -1 },
		"group w/o code": func(e *VodEdit) { e.PlayGroups = []VodPlayGroup{{From: " ", Text: "a$https://x"}} },
		"# in address":   func(e *VodEdit) { e.PlayGroups = []VodPlayGroup{{From: "x", Text: "a$https://x#y"}} },
	}
	for name, mutate := range cases {
		v := sampleVod()
		e := vodEditOf(v)
		mutate(&e)
		if err := e.applyTo(&v); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

// 采集数据的各种形状: 地址含 $、没有集名、播放器代码比播放组少、末尾多余的 #
func TestVodEditRoundTripRealShapes(t *testing.T) {
	v := sampleVod()
	v.VodPlayFrom = "ffm3u8"
	v.VodPlayUrl = "第1集$https://x/a.m3u8?sig=$abc#https://x/b.m3u8$$$正片$https://y/1.m3u8#"
	e := vodEditOf(v)
	if len(e.PlayGroups) != 2 || e.PlayGroups[0].Text != "第1集$https://x/a.m3u8?sig=$abc\nhttps://x/b.m3u8" || e.PlayGroups[1].From != "play2" {
		t.Fatalf("play groups = %+v", e.PlayGroups)
	}
	got := v
	if err := e.applyTo(&got); err != nil {
		t.Fatal(err)
	}
	if got.VodPlayUrl != "第1集$https://x/a.m3u8?sig=$abc#https://x/b.m3u8$$$正片$https://y/1.m3u8" || got.VodPlayFrom != "ffm3u8$$$play2" {
		t.Fatalf("play = %q / %q", got.VodPlayFrom, got.VodPlayUrl)
	}
}

// 打开编辑页之后采集又更新了影片: 保存被拒绝, 不覆盖新采集的数据
func TestVodEditRejectsStaleCollectedData(t *testing.T) {
	v := sampleVod()
	e := vodEditOf(v)
	v.VodPlayUrl += "#第03集$https://a/3.m3u8"
	v.VodRemarks = "更新至3集"
	if err := e.applyTo(&v); err == nil || !strings.Contains(err.Error(), "重新载入") {
		t.Fatalf("want a reload error, got %v", err)
	}
}
