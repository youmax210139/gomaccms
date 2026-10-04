package maccms

import (
	"fmt"
	"html/template"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

// Ctx 一次页面渲染的上下文 (public.RenderPage 放在模板数据的 CtxKey 中), 数据标签按它取当前方案、语言与页码
type Ctx struct {
	SchemeId int64
	Lang     string
	Page     int   // 当前页码 (paging="yes" 的标签使用)
	TypeId   int64 // 当前页面的分类 (type="current")
	BaseURL  string
}

// 模板数据中的键
const (
	CtxKey    = "maccmsCtx"  // Ctx
	GlobalKey = "maccms"     // {$maccms.xxx}
	PagingKey = "__PAGING__" // {$__PAGING__.xxx}
)

// Item 标签输出的一项: 栏位名 → 文字 (与 MacCMS 的栏位名一致, 如 vod_name)
type Item = map[string]string

// Result 标签的结果; Total 为符合条件的总数 (分页用)
type Result struct {
	List  []Item
	Total int
}

// Handler 一个数据标签的实现 ({maccms:<name>}); 只做业务查询, 参数已校验
type Handler func(ctx Ctx, opts QueryOptions) (Result, error)

var (
	mu       sync.RWMutex
	handlers = map[string]Handler{}
)

// Register 注册数据标签 (启动时由 tags.Register 调用; 重复注册时覆盖, 便于扩充或测试替换)
func Register(name string, h Handler) {
	mu.Lock()
	defer mu.Unlock()
	handlers[name] = h
}

func lookup(name string) (Handler, bool) {
	mu.RLock()
	defer mu.RUnlock()
	h, ok := handlers[name]
	return h, ok
}

// Funcs 翻译后的模板使用的函数 (renderer 合并到主题的 FuncMap)
func Funcs() template.FuncMap {
	return template.FuncMap{
		"maccmsTag":   Tag,
		"maccmsGet":   Get,
		"maccmsEmpty": Empty,
		"maccmsList":  List,
		"maccmsCmp":   Cmp,
	}
}

// Tag {maccms:name ...}: 校验参数后交给注册的 Handler; paging="yes" 时把分页信息写入模板数据的 __PAGING__.
// kv 为 参数名, 值, 参数名, 值 ...; 出错时返回 error (模板停止渲染, 不 panic)
func Tag(data map[string]any, name string, kv ...any) ([]Item, error) {
	h, ok := lookup(name)
	if !ok {
		return nil, fmt.Errorf("maccms: 未注册的标签 %q", name)
	}
	if len(kv)%2 != 0 {
		return nil, fmt.Errorf("maccms:%s 参数不成对", name)
	}
	attrs := make(map[string]string, len(kv)/2)
	for i := 0; i < len(kv); i += 2 {
		attrs[fmt.Sprint(kv[i])] = fmt.Sprint(kv[i+1])
	}
	opts, err := ParseOptions(attrs)
	if err != nil {
		return nil, fmt.Errorf("maccms:%s %w", name, err)
	}
	ctx, _ := data[CtxKey].(Ctx)
	page := 1
	if opts.Paging {
		page = min(max(ctx.Page, 1), MaxPage)
	}
	opts.Offset = opts.Start - 1 + (page-1)*opts.Num
	res, err := h(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("maccms:%s %w", name, err)
	}
	if opts.Paging && data != nil {
		pages := (res.Total + opts.Num - 1) / opts.Num
		data[PagingKey] = map[string]any{"record_total": res.Total, "page_current": page, "page_total": max(pages, 1), "page_size": opts.Num}
	}
	return res.List, nil
}

// Get 依次按键取值 ({$vo.vod_name}; map 的键或结构体的栏位), 任何一层不存在时为空字串
func Get(v any, keys ...string) any {
	for _, k := range keys {
		switch m := v.(type) {
		case map[string]any:
			v = m[k]
		case map[string]string:
			v = m[k]
		default:
			rv := reflect.Indirect(reflect.ValueOf(v))
			if rv.Kind() == reflect.Struct { // 结构体栏位, 名称不分大小写 ({$seo.title})
				f := rv.FieldByNameFunc(func(n string) bool { return strings.EqualFold(n, k) })
				if !f.IsValid() || !f.CanInterface() {
					return ""
				}
				v = f.Interface()
				continue
			}
			if rv.Kind() != reflect.Map || rv.Type().Key().Kind() != reflect.String {
				return ""
			}
			e := rv.MapIndex(reflect.ValueOf(k).Convert(rv.Type().Key()))
			if !e.IsValid() {
				return ""
			}
			v = e.Interface()
		}
		if v == nil {
			return ""
		}
	}
	return v
}

// Empty MacCMS 的空值: nil、空字串、"0"、0、false、空的列表或 map
func Empty(v any) bool {
	if v == nil {
		return true
	}
	if s, ok := v.(string); ok {
		return s == "" || s == "0"
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice, reflect.Map, reflect.Array:
		return rv.Len() == 0
	case reflect.Bool:
		return !rv.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return rv.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return rv.Float() == 0
	}
	return false
}

// List {foreach} 的来源: 列表或 map 原样返回, 其他值 (含空值) 为空列表, 不会让 range 出错
func List(v any) any {
	if v == nil {
		return []any{}
	}
	switch reflect.ValueOf(v).Kind() {
	case reflect.Slice, reflect.Array, reflect.Map:
		return v
	}
	return []any{}
}

// Cmp {if condition} 中的比较: 两边都是数字时按数字, 否则按字串
func Cmp(op string, a, b any) (bool, error) {
	as, bs := fmt.Sprint(a), fmt.Sprint(b)
	af, aerr := strconv.ParseFloat(strings.TrimSpace(as), 64)
	bf, berr := strconv.ParseFloat(strings.TrimSpace(bs), 64)
	numeric := aerr == nil && berr == nil
	c := strings.Compare(as, bs)
	if numeric {
		c = 0
		if af < bf {
			c = -1
		} else if af > bf {
			c = 1
		}
	}
	switch op {
	case "eq":
		return c == 0, nil
	case "neq":
		return c != 0, nil
	case "gt":
		return c > 0, nil
	case "lt":
		return c < 0, nil
	case "egt":
		return c >= 0, nil
	case "elt":
		return c <= 0, nil
	}
	return false, fmt.Errorf("maccms: 不支持的比较 %q", op)
}
