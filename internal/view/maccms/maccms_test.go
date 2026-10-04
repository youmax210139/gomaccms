package maccms

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeVods 测试用的影片数据; 记录最后一次收到的参数
var lastOpts QueryOptions

func init() {
	Register("vod", func(ctx Ctx, o QueryOptions) (Result, error) {
		lastOpts = o
		var all []Item
		for i := 1; i <= 25; i++ {
			all = append(all, Item{"vod_id": fmt.Sprint(i), "vod_name": fmt.Sprintf("影片%d", i), "vod_pic": fmt.Sprintf("/p/%d.jpg", i), "vod_url": fmt.Sprintf("/filmDetail?link=%d", i)})
		}
		end := min(o.Offset+o.Num, len(all))
		if o.Offset >= len(all) {
			return Result{Total: len(all)}, nil
		}
		return Result{List: all[o.Offset:end], Total: len(all)}, nil
	})
	Register("type", func(ctx Ctx, o QueryOptions) (Result, error) {
		list := []Item{{"type_id": "1", "type_name": "电影", "type_url": "/filmClassify?Pid=1"}, {"type_id": "2", "type_name": "电视剧", "type_url": "/filmClassify?Pid=2"}}
		return Result{List: list[:min(o.Num, len(list))], Total: len(list)}, nil
	})
}

// render 翻译 + 解析 + 执行; includes 为可被 include 的模板
func render(t *testing.T, src string, data map[string]any, includes map[string]string) (string, error) {
	t.Helper()
	out, err := Translate(src, func(name string) (string, error) {
		s, ok := includes[name]
		if !ok {
			return "", fmt.Errorf("not found")
		}
		return s, nil
	})
	if err != nil {
		return "", err
	}
	tmpl, err := template.New("t").Funcs(Funcs()).Parse(out)
	if err != nil {
		return "", fmt.Errorf("parse: %w\n%s", err, out)
	}
	if data == nil {
		data = map[string]any{}
	}
	var b bytes.Buffer
	err = tmpl.Execute(&b, data)
	return b.String(), err
}

// squash 去掉空白, 方便比对
func squash(s string) string {
	return strings.Join(strings.Fields(s), "")
}

func mustRender(t *testing.T, src string, data map[string]any, includes map[string]string) string {
	t.Helper()
	out, err := render(t, src, data, includes)
	if err != nil {
		t.Fatal(err)
	}
	return squash(out)
}

func TestVodTag(t *testing.T) {
	got := mustRender(t, `{maccms:vod num="3" type="all" order="desc" by="time"}[{$vo.vod_name}|{$vo.vod_pic}|{$vo.vod_url}]{/maccms:vod}`, nil, nil)
	want := "[影片1|/p/1.jpg|/filmDetail?link=1][影片2|/p/2.jpg|/filmDetail?link=2][影片3|/p/3.jpg|/filmDetail?link=3]"
	if got != want {
		t.Fatalf("got %s\nwant %s", got, want)
	}
	if lastOpts.Num != 3 || lastOpts.Order != "desc" || lastOpts.By != "time" || lastOpts.Type != "all" {
		t.Fatalf("options = %+v", lastOpts)
	}
}

func TestTypeTagAndCustomVar(t *testing.T) {
	got := mustRender(t, `{maccms:type num="10" id="t" key="i"}{$i}:{$t.type_name}@{$t.type_url};{/maccms:type}`, nil, nil)
	if got != "0:电影@/filmClassify?Pid=1;1:电视剧@/filmClassify?Pid=2;" {
		t.Fatalf("got %s", got)
	}
}

func TestGlobalsAndEscaping(t *testing.T) {
	data := map[string]any{GlobalKey: map[string]any{"site_name": "<b>金丝雀</b>", "site_url": "https://x.test"}}
	got := mustRender(t, `{$maccms.site_name}|{$maccms.site_url}|{$maccms.missing}|{$nothing.at.all}`, data, nil)
	if got != "&lt;b&gt;金丝雀&lt;/b&gt;|https://x.test||" {
		t.Fatalf("got %s", got)
	}
}

func TestInclude(t *testing.T) {
	inc := map[string]string{"public/head": `<h1>{$maccms.site_name}</h1>{include file="public/nav"}`, "public/nav": `{maccms:type num="1"}<a>{$vo.type_name}</a>{/maccms:type}`}
	got := mustRender(t, `{include file="public/head.html"}<p>body</p>`, map[string]any{GlobalKey: map[string]any{"site_name": "站"}}, inc)
	if got != "<h1>站</h1><a>电影</a><p>body</p>" {
		t.Fatalf("got %s", got)
	}
	if _, err := render(t, `{include file="../secret"}`, nil, inc); err == nil {
		t.Fatal("include outside the template directory must fail")
	}
	loop := map[string]string{"a": `{include file="a"}`}
	if _, err := render(t, `{include file="a"}`, nil, loop); err == nil || !strings.Contains(err.Error(), "嵌套") {
		t.Fatalf("recursive include must fail, got %v", err)
	}
}

func TestIfElse(t *testing.T) {
	src := `{maccms:vod num="4"}{if condition="$vo.vod_id eq 1"}first{elseif condition="$vo.vod_id > 2 && $vo.vod_name != ''"/}big{else/}mid{/if};{/maccms:vod}`
	if got := mustRender(t, src, nil, nil); got != "first;mid;big;big;" {
		t.Fatalf("got %s", got)
	}
	data := map[string]any{"user": map[string]any{"name": "a", "level": "3"}}
	if got := mustRender(t, `{if condition="!($user.level < 2) and ($user.name == 'a' or $user.x)"}ok{/if}`, data, nil); got != "ok" {
		t.Fatalf("got %s", got)
	}
}

func TestEmptyAndForeach(t *testing.T) {
	data := map[string]any{"list": []map[string]string{{"n": "a"}, {"n": "b"}}, "none": []string{}}
	src := `{empty name="none"}无{/empty}{notempty name="list"}有{/notempty}{foreach name="list" item="x" key="k"}{$k}{$x.n}{/foreach}|{foreach $list as $v}{$v.n}{/foreach}|{foreach $missing as $v}x{/foreach}`
	if got := mustRender(t, src, data, nil); got != "无有0a1b|ab|" {
		t.Fatalf("got %s", got)
	}
}

func TestPaging(t *testing.T) {
	data := map[string]any{CtxKey: Ctx{Page: 3}}
	got := mustRender(t, `{maccms:vod num="10" paging="yes"}{$vo.vod_id},{/maccms:vod}|{$__PAGING__.record_total}/{$__PAGING__.page_current}/{$__PAGING__.page_total}`, data, nil)
	if got != "21,22,23,24,25,|25/3/3" {
		t.Fatalf("got %s", got)
	}
	// 页码上限; start 在分页时作为每页的偏移起点
	data = map[string]any{CtxKey: Ctx{Page: 99999}}
	mustRender(t, `{maccms:vod num="10" paging="yes"}{/maccms:vod}`, data, nil)
	if lastOpts.Offset != (MaxPage-1)*10 {
		t.Fatalf("page must be capped at %d, offset = %d", MaxPage, lastOpts.Offset)
	}
	mustRender(t, `{maccms:vod num="5" start="3"}{/maccms:vod}`, nil, nil)
	if lastOpts.Offset != 2 {
		t.Fatalf("start=3 → offset 2, got %d", lastOpts.Offset)
	}
}

func TestIllegalParams(t *testing.T) {
	for _, src := range []string{
		`{maccms:vod num="abc"}{/maccms:vod}`,
		`{maccms:vod num="-1"}{/maccms:vod}`,
		`{maccms:vod order="random"}{/maccms:vod}`,
		`{maccms:vod by="vod_name"}{/maccms:vod}`,
		`{maccms:vod year="20x9"}{/maccms:vod}`,
		`{maccms:vod letter="AB"}{/maccms:vod}`,
		`{maccms:vod ids="1,a"}{/maccms:vod}`,
		`{maccms:vod foo="1"}{/maccms:vod}`,
		`{maccms:nosuchtag}{/maccms:nosuchtag}`,
	} {
		if _, err := render(t, src, nil, nil); err == nil {
			t.Errorf("%s: want an error", src)
		}
	}
	// 超过上限的数字取上限
	mustRender(t, `{maccms:vod num="100000" start="99999"}{/maccms:vod}`, nil, nil)
	if lastOpts.Num != MaxNum || lastOpts.Start != MaxStart {
		t.Fatalf("num/start must be capped, got %+v", lastOpts)
	}
}

func TestSQLInjectionRejected(t *testing.T) {
	for _, v := range []string{`' OR 1=1 --`, `动作'; DROP TABLE vod; --`, `a" OR "1"="1`, `1) UNION SELECT`} {
		if _, err := ParseOptions(map[string]string{"class": v}); err == nil {
			t.Errorf("class=%q must be rejected", v)
		}
	}
	if _, err := ParseOptions(map[string]string{"type": "1 OR 1=1"}); err == nil {
		t.Error("type must only take IDs")
	}
	if _, err := ParseOptions(map[string]string{"by": "vod_id; DROP TABLE vod"}); err == nil {
		t.Error("by must be whitelisted")
	}
	if o, err := ParseOptions(map[string]string{"class": "动作 科幻", "area": "中国大陆", "type": "1,2"}); err != nil || o.Class != "动作 科幻" || len(o.TypeIds) != 2 {
		t.Fatalf("valid values rejected: %+v %v", o, err)
	}
}

func TestParseErrors(t *testing.T) {
	for _, src := range []string{
		`{maccms:vod}`,                          // 没有结束
		`{/maccms:vod}`,                         // 多余的结束
		`{if condition="$a eq 1"}{/maccms:vod}`, // 不匹配
		`{if condition="$a ++ 1"}x{/if}`,        // 条件无法识别
		`{if condition="system('x')"}x{/if}`,    // 不支持的函数
		`{foreach name="a b"}{/foreach}`,
		`{elseif condition="1"/}`,
	} {
		if _, err := Translate(src, nil); err == nil {
			t.Errorf("%s: want a parse error", src)
		}
	}
	_, err := Translate("line1\nline2\n{/if}", nil)
	if err == nil || !strings.Contains(err.Error(), "第 3 行") {
		t.Fatalf("error must name the line, got %v", err)
	}
}

// 现有主题模板 (Go 模板、CSS、JS 的大括号) 不受影响: 没用 MacCMS 标签的翻译前后完全相同, 用了的 (播放页广告位) 能正常翻译
func TestPlainTemplatesUnchanged(t *testing.T) {
	files, _ := filepath.Glob("../../../themes/default/templates/*/*.html")
	if len(files) == 0 {
		t.Skip("default theme not found")
	}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		out, err := Translate(string(b), nil)
		if err != nil {
			t.Errorf("%s: %v", f, err)
		} else if !strings.Contains(string(b), "{maccms:") && out != string(b) {
			// 没有用 MacCMS 标签的模板必须原样不变
			t.Errorf("%s changed by Translate", f)
		}
	}
	src := `<style>a{color:red}</style><script>if(x){y()}else{z()} const s=` + "`${a}`" + `</script>{{.Title}}`
	if out, _ := Translate(src, nil); out != src {
		t.Fatalf("plain text changed:\n%s", out)
	}
}

// {$seo.title}: 结构体栏位按名称 (不分大小写) 读取
func TestGetStructField(t *testing.T) {
	type pageSEO struct{ Title, Keywords string }
	data := map[string]any{"seo": pageSEO{Title: "标题 <x>", Keywords: "k"}, "p": &pageSEO{Title: "ptr"}}
	got := mustRender(t, `{$seo.title}|{$seo.Keywords}|{$seo.missing}|{$p.title}`, data, nil)
	if got != "标题&lt;x&gt;|k||ptr" {
		t.Fatalf("got %s", got)
	}
}

// slot 参数 (广告位): 小写字母、数字、下划线
func TestSlotOption(t *testing.T) {
	if o, err := ParseOptions(map[string]string{"slot": "play_top"}); err != nil || o.Slot != "play_top" {
		t.Fatalf("slot = %+v %v", o, err)
	}
	for _, v := range []string{"Play", "a b", "x'--", "a;b"} {
		if _, err := ParseOptions(map[string]string{"slot": v}); err == nil {
			t.Errorf("slot=%q must be rejected", v)
		}
	}
}
