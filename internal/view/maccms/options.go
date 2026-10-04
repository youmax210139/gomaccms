package maccms

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// 数据标签参数的上限
const (
	DefaultNum = 10
	MaxNum     = 100  // 每次最多取的条数
	MaxStart   = 1000 // start 最大值
	MaxPage    = 1000 // 分页页码最大值
	MaxIds     = 100  // ids / type 最多几个ID
	maxText    = 30   // 文字筛选值的最大长度
)

// QueryOptions 数据标签统一的查询参数 (已按白名单校验)
type QueryOptions struct {
	Num     int
	Start   int     // 从第几条开始 (1 起)
	Ids     []int64 // 指定ID
	Type    string  // "all" (默认) / "current" (当前页的分类) / "ids" (见 TypeIds)
	TypeIds []int64
	Class   string
	Tag     string
	Area    string
	Lang    string
	Year    string // 2019 或 2010-2020
	Letter  string // A-Z / 0-9
	State   string
	Version string
	Order   string // asc / desc
	By      string // time / time_add / hits / score / id / sort
	Paging  bool
	Slot    string // 广告位 ({maccms:ad})
	Offset  int    // 由 Tag 按 start、分页计算
}

// allowedAttrs 数据标签可以使用的参数 (id / key 为模板中的变量名, 由解析器处理)
var allowedAttrs = map[string]bool{"num": true, "ids": true, "type": true, "class": true, "tag": true, "area": true,
	"lang": true, "year": true, "letter": true, "state": true, "version": true, "order": true, "by": true,
	"start": true, "paging": true, "slot": true}

// allowedBy 排序栏位白名单
var allowedBy = map[string]bool{"time": true, "time_add": true, "hits": true, "score": true, "id": true, "sort": true}

var (
	yearPattern   = regexp.MustCompile(`^[12]\d{3}(-[12]\d{3})?$`)
	letterPattern = regexp.MustCompile(`^[A-Z0-9]$`)
	slotPattern   = regexp.MustCompile(`^[a-z0-9_]{1,30}$`)
	// 文字筛选值: 文字、数字、空格与少数标点, 不允许引号、分号、注释符等
	textPattern = regexp.MustCompile(`^[\p{L}\p{N} ·_/+\-]+$`)
)

// ParseOptions 校验并转换数据标签的参数; 不认识的参数或不合法的值返回错误
func ParseOptions(attrs map[string]string) (QueryOptions, error) {
	o := QueryOptions{Num: DefaultNum, Start: 1, Type: "all", Order: "desc", By: "time"}
	for k, raw := range attrs {
		v := strings.TrimSpace(raw)
		if !allowedAttrs[k] {
			return o, fmt.Errorf("不支持的参数 %s", k)
		}
		if v == "" {
			continue
		}
		var err error
		switch k {
		case "num":
			o.Num, err = boundedInt(k, v, MaxNum)
		case "start":
			o.Start, err = boundedInt(k, v, MaxStart)
		case "ids":
			o.Ids, err = idList(k, v)
		case "type":
			if v == "all" || v == "current" {
				o.Type = v
			} else if o.TypeIds, err = idList(k, v); err == nil {
				o.Type = "ids"
			}
		case "order":
			if v != "asc" && v != "desc" {
				err = fmt.Errorf("order 只能是 asc 或 desc")
			}
			o.Order = v
		case "by":
			if !allowedBy[v] {
				err = fmt.Errorf("by 不支持 %q", v)
			}
			o.By = v
		case "year":
			if !yearPattern.MatchString(v) {
				err = fmt.Errorf("year 格式应为 2019 或 2010-2020")
			}
			o.Year = v
		case "letter":
			v = strings.ToUpper(v)
			if !letterPattern.MatchString(v) {
				err = fmt.Errorf("letter 只能是一个字母或数字")
			}
			o.Letter = v
		case "slot":
			if !slotPattern.MatchString(v) {
				err = fmt.Errorf("slot 只能是小写字母、数字与下划线")
			}
			o.Slot = v
		case "paging":
			if v != "yes" && v != "no" {
				err = fmt.Errorf("paging 只能是 yes 或 no")
			}
			o.Paging = v == "yes"
		default: // class / tag / area / lang / state / version
			if utf8.RuneCountInString(v) > maxText || !textPattern.MatchString(v) {
				err = fmt.Errorf("%s 的值不合法", k)
			}
			setText(&o, k, v)
		}
		if err != nil {
			return o, err
		}
	}
	return o, nil
}

func setText(o *QueryOptions, k, v string) {
	switch k {
	case "class":
		o.Class = v
	case "tag":
		o.Tag = v
	case "area":
		o.Area = v
	case "lang":
		o.Lang = v
	case "state":
		o.State = v
	case "version":
		o.Version = v
	}
}

// boundedInt 正整数, 超过上限时取上限
func boundedInt(name, v string, max int) (int, error) {
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("%s 必须是正整数", name)
	}
	return min(n, max), nil
}

// idList 逗号分隔的正整数ID, 最多 MaxIds 个
func idList(name, v string) ([]int64, error) {
	parts := strings.Split(v, ",")
	if len(parts) > MaxIds {
		return nil, fmt.Errorf("%s 最多 %d 个", name, MaxIds)
	}
	ids := make([]int64, 0, len(parts))
	for _, p := range parts {
		id, err := strconv.ParseInt(strings.TrimSpace(p), 10, 64)
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("%s 只能是逗号分隔的ID", name)
		}
		ids = append(ids, id)
	}
	return ids, nil
}
