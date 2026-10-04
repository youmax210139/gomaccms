package seo

import "testing"

func TestResolve(t *testing.T) {
	ctx := Context{
		Site:    Site{Name: "金丝雀", URL: "https://x.test", Keywords: "电影", Description: "在线观影"},
		Page:    2,
		Type:    &Type{Name: "动作片"},
		Vod:     &Vod{Name: "流浪地球", Year: "2019", Content: "<p>  太阳\n即将毁灭 </p>"},
		Episode: &Episode{Name: "第01集", Index: 1},
	}
	got := Resolve("{vod_name}({vod_year}) {episode_name}#{episode_index} - {type_name} - 第{page}页 - {site_name} {unknown}{vod_content}", ctx)
	if got != "流浪地球(2019) 第01集#1 - 动作片 - 第2页 - 金丝雀 太阳 即将毁灭" {
		t.Fatalf("got %q", got)
	}
	// 没有影片 / 分类时对应的占位符为空, 不 panic
	if got := Resolve("{vod_name}{type_name}{episode_name}|{site_name}", Context{Site: Site{Name: "站"}}); got != "|站" {
		t.Fatalf("got %q", got)
	}
	if got := Resolve("{page} {year}", Context{}); got != "1 "+currentYear() {
		t.Fatalf("page defaults to 1, got %q", got)
	}
}

func TestBuildFallback(t *testing.T) {
	ctx := Context{PageType: PageType, Site: Site{Name: "站"}, Type: &Type{Name: "动作片"}}
	site := Text{Title: "站点标题", Keywords: "站点关键字", Description: "站点描述"}
	// 规则 → 站点, 每个栏位各自回退
	got := Build(ctx, Text{Title: "{type_name}大全 - {site_name}", Keywords: "{type_name},规则关键字"}, site)
	if got != (Text{Title: "动作片大全 - 站", Keywords: "动作片,规则关键字", Description: "站点描述"}) {
		t.Fatalf("got %+v", got)
	}
	// 没有规则时分类页标题用内置的默认规则 (只含占位符)
	if got := Build(ctx, Text{}, site); got.Title != "动作片 - 站" || got.Keywords != "站点关键字" {
		t.Fatalf("default rule: %+v", got)
	}
	// 规则替换后为空 (占位符都没有值) 时回退到站点
	if got := Build(Context{PageType: PageHome}, Text{Title: "{vod_name}"}, site); got.Title != "站点标题" {
		t.Fatalf("empty result must fall back: %+v", got)
	}
	// 已删除的分类 SEO 占位符替换为空
	if got := Resolve("{type_name}{type_title}{type_key}{type_des}", ctx); got != "动作片" {
		t.Fatalf("removed placeholders must be empty, got %q", got)
	}
}

func TestNormalizeRules(t *testing.T) {
	rules := Rules{PageType: {"zh-CN": {Title: " {type_name} "}, "vi": {}}, "bogus": {"zh-CN": {Title: "x"}}}
	list, err := rows(3, rules)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].SchemeId != 3 || list[0].PageType != PageType || list[0].Lang != "zh-CN" || list[0].Title != "{type_name}" {
		t.Fatalf("rows = %+v", list)
	}
	long := make([]byte, 600)
	for i := range long {
		long[i] = 'a'
	}
	if _, err := rows(3, Rules{PageHome: {"zh-CN": {Description: string(long)}}}); err == nil {
		t.Fatal("too long description must be rejected")
	}
}

// 关键字等以逗号分隔的规则: 占位符为空时不留下多余的逗号
func TestResolveCleansSeparators(t *testing.T) {
	ctx := Context{Site: Site{Name: "站", Keywords: "电影, 电视剧"}, Vod: &Vod{Name: "流浪地球"}}
	if got := Resolve("{vod_name},{vod_actor},{vod_director},{site_keywords}", ctx); got != "流浪地球,电影, 电视剧" {
		t.Fatalf("got %q", got)
	}
	if got := Resolve("{vod_actor}，{vod_name}，", ctx); got != "流浪地球" {
		t.Fatalf("full-width commas: got %q", got)
	}
	if got := Resolve("{vod_actor},{vod_director}", ctx); got != "" {
		t.Fatalf("only separators left must become empty, got %q", got)
	}
}

func TestResolveDropsEmptyParens(t *testing.T) {
	if got := Resolve("{vod_name}({vod_year}) - {site_name}", Context{Site: Site{Name: "站"}, Vod: &Vod{Name: "片"}}); got != "片 - 站" {
		t.Fatalf("got %q", got)
	}
	if got := Resolve("{vod_name}（{vod_year}）", Context{Vod: &Vod{Name: "片", Year: "2019"}}); got != "片（2019）" {
		t.Fatalf("got %q", got)
	}
}
