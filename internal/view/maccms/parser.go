package maccms

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Translate 把模板中的 MacCMS 标签翻译成 Go html/template 语法 (只做文字转换, 不查数据); 不是 MacCMS 标签的
// 文字 (含原有的 {{ }} 与 CSS / JS 的大括号) 原样保留. include 读取 {include file="..."} 的模板 (相对模板目录, 已去掉 .html).
//
// 支持: {$a.b} / {maccms:name ...}...{/maccms:name} / {include file=""} / {if condition=""}{elseif condition=""/}{else/}{/if}
// / {empty name=""}...{/empty} / {notempty name=""}...{/notempty} / {foreach name="" item="" key=""} 或 {foreach $a as $k=>$v}...{/foreach}
func Translate(src string, include func(name string) (string, error)) (string, error) {
	t := &translator{include: include}
	out, err := t.run(src, 0)
	if err != nil {
		return "", err
	}
	if len(t.stack) > 0 {
		return "", fmt.Errorf("maccms: {%s} 没有结束标签", t.stack[len(t.stack)-1].name)
	}
	return out, nil
}

const maxIncludeDepth = 5

// tagPattern MacCMS 标签; 参数值中可以有 {$var}
var tagPattern = regexp.MustCompile(`\{(\$[A-Za-z_][\w.]*|/?maccms:[a-z]+(?:[^{}]|\{\$[\w.]+\})*|include\s[^{}]*|(?:else)?if\s(?:[^{}]|\{\$[\w.]+\})*|else\s*/?|/if|(?:not)?empty\s[^{}]*|/(?:not)?empty|foreach\s[^{}]*|/foreach)\}`)

var (
	attrPattern      = regexp.MustCompile(`([a-z_]+)\s*=\s*"([^"]*)"`)
	includePattern   = regexp.MustCompile(`^[\w\-/]+$`)
	foreachAsPattern = regexp.MustCompile(`^foreach\s+\$([\w.]+)\s+as\s+\$(\w+)(?:\s*=>\s*\$(\w+))?\s*$`)
	identPattern     = regexp.MustCompile(`^[A-Za-z_]\w*$`)
)

type block struct {
	name string   // maccms:vod / if / empty / notempty / foreach
	vars []string // 这个区块声明的变量
}

type translator struct {
	include func(string) (string, error)
	stack   []block
}

// goActionPattern 原有的 Go 模板动作 {{ ... }}, 其中的文字不当作 MacCMS 标签 (如 {{if .site}})
var goActionPattern = regexp.MustCompile(`(?s)\{\{.*?\}\}`)

func (t *translator) run(src string, depth int) (string, error) {
	var b strings.Builder
	last := 0
	actions := goActionPattern.FindAllStringIndex(src, -1)
	for _, m := range tagPattern.FindAllStringSubmatchIndex(src, -1) {
		if insideAny(actions, m[0], m[1]) {
			continue
		}
		b.WriteString(src[last:m[0]])
		last = m[1]
		body := src[m[2]:m[3]]
		out, err := t.tag(body, depth)
		if err != nil {
			line := strings.Count(src[:m[0]], "\n") + 1
			return "", fmt.Errorf("maccms: 第 %d 行 {%s}: %w", line, body, err)
		}
		b.WriteString(out)
	}
	b.WriteString(src[last:])
	return b.String(), nil
}

// insideAny [start, end) 是否与某个 Go 模板动作重叠
func insideAny(spans [][]int, start, end int) bool {
	for _, s := range spans {
		if start < s[1] && end > s[0] {
			return true
		}
	}
	return false
}

func (t *translator) tag(body string, depth int) (string, error) {
	switch {
	case strings.HasPrefix(body, "$"):
		return "{{" + t.varExpr(body[1:]) + "}}", nil
	case strings.HasPrefix(body, "maccms:"):
		return t.dataTag(body)
	case strings.HasPrefix(body, "/maccms:"):
		return t.end("maccms:" + strings.TrimSpace(body[len("/maccms:"):]))
	case strings.HasPrefix(body, "include"):
		return t.includeFile(body, depth)
	case strings.HasPrefix(body, "if"):
		cond, err := t.condition(body)
		if err != nil {
			return "", err
		}
		t.push("if", nil)
		return "{{if " + cond + "}}", nil
	case strings.HasPrefix(body, "elseif"):
		if !t.inside("if") {
			return "", fmt.Errorf("elseif 不在 if 中")
		}
		cond, err := t.condition(body)
		if err != nil {
			return "", err
		}
		return "{{else if " + cond + "}}", nil
	case strings.HasPrefix(body, "else"):
		if len(t.stack) == 0 {
			return "", fmt.Errorf("else 不在区块中")
		}
		return "{{else}}", nil
	case body == "/if":
		return t.end("if")
	case strings.HasPrefix(body, "empty"), strings.HasPrefix(body, "notempty"):
		name := attrs(body)["name"]
		if !identPathOK(name) {
			return "", fmt.Errorf("name 不合法")
		}
		kind := strings.Fields(body)[0]
		t.push(kind, nil)
		if kind == "empty" {
			return "{{if maccmsEmpty " + t.varExpr(name) + "}}", nil
		}
		return "{{if not (maccmsEmpty " + t.varExpr(name) + ")}}", nil
	case body == "/empty", body == "/notempty":
		return t.end(body[1:])
	case strings.HasPrefix(body, "foreach"):
		return t.foreach(body)
	case body == "/foreach":
		return t.end("foreach")
	}
	return "", fmt.Errorf("无法识别")
}

// dataTag {maccms:vod num="10" ... id="vo" key="key"} → {{range $key, $vo := maccmsTag $ "vod" "num" "10" ...}}
func (t *translator) dataTag(body string) (string, error) {
	name, rest, _ := strings.Cut(strings.TrimPrefix(body, "maccms:"), " ")
	a := attrs(rest)
	item, key := orDefault(a["id"], "vo"), orDefault(a["key"], "key")
	if !identPattern.MatchString(item) || !identPattern.MatchString(key) {
		return "", fmt.Errorf("id / key 只能是变量名")
	}
	delete(a, "id")
	delete(a, "key")
	var args strings.Builder
	for _, m := range attrPattern.FindAllStringSubmatch(rest, -1) { // 按原顺序输出, 便于阅读
		if _, ok := a[m[1]]; ok {
			args.WriteString(" " + strconv.Quote(m[1]) + " " + t.value(m[2]))
		}
	}
	t.push("maccms:"+name, []string{item, key})
	return fmt.Sprintf("{{range $%s, $%s := maccmsTag $ %s%s}}", key, item, strconv.Quote(name), args.String()), nil
}

// foreach {foreach name="list" item="vo" key="key"} 或 {foreach $list as $key=>$vo}
func (t *translator) foreach(body string) (string, error) {
	var src, key, item string
	if m := foreachAsPattern.FindStringSubmatch(body); m != nil {
		src, key, item = m[1], m[2], m[3]
		if item == "" { // {foreach $list as $vo}
			key, item = "key", m[2]
		}
	} else {
		a := attrs(body)
		src, item, key = a["name"], orDefault(a["item"], "vo"), orDefault(a["key"], "key")
	}
	if !identPathOK(src) || !identPattern.MatchString(item) || !identPattern.MatchString(key) {
		return "", fmt.Errorf("foreach 参数不合法")
	}
	expr := t.varExpr(src)
	t.push("foreach", []string{item, key})
	return fmt.Sprintf("{{range $%s, $%s := maccmsList %s}}", key, item, expr), nil
}

// includeFile 读入并翻译被包含的模板 (最多嵌套 maxIncludeDepth 层)
func (t *translator) includeFile(body string, depth int) (string, error) {
	file := strings.TrimSuffix(attrs(body)["file"], ".html")
	if !includePattern.MatchString(file) || strings.Contains(file, "..") {
		return "", fmt.Errorf("include 的 file 不合法")
	}
	if depth >= maxIncludeDepth {
		return "", fmt.Errorf("include 嵌套超过 %d 层", maxIncludeDepth)
	}
	if t.include == nil {
		return "", fmt.Errorf("不支持 include")
	}
	src, err := t.include(file)
	if err != nil {
		return "", fmt.Errorf("include %s: %w", file, err)
	}
	return t.run(src, depth+1)
}

func (t *translator) push(name string, vars []string) {
	t.stack = append(t.stack, block{name: name, vars: vars})
}

func (t *translator) inside(name string) bool {
	return len(t.stack) > 0 && t.stack[len(t.stack)-1].name == name
}

func (t *translator) end(name string) (string, error) {
	if !t.inside(name) {
		if len(t.stack) == 0 {
			return "", fmt.Errorf("多余的结束标签")
		}
		return "", fmt.Errorf("结束标签与 {%s} 不匹配", t.stack[len(t.stack)-1].name)
	}
	t.stack = t.stack[:len(t.stack)-1]
	return "{{end}}", nil
}

// declared 变量是否由外层的 {maccms:...} / {foreach} 声明
func (t *translator) declared(name string) bool {
	for _, b := range t.stack {
		for _, v := range b.vars {
			if v == name {
				return true
			}
		}
	}
	return false
}

// varExpr a.b.c → (maccmsGet $a "b" "c") (a 为循环变量) 或 (maccmsGet $ "a" "b" "c") (a 为页面数据)
func (t *translator) varExpr(path string) string {
	parts := strings.Split(path, ".")
	var b strings.Builder
	if t.declared(parts[0]) {
		b.WriteString("(maccmsGet $" + parts[0])
		parts = parts[1:]
	} else {
		b.WriteString("(maccmsGet $")
	}
	for _, p := range parts {
		if p != "" {
			b.WriteString(" " + strconv.Quote(p))
		}
	}
	return b.String() + ")"
}

// value 参数值: {$a.b} 或 $a.b 为变量, 其余为字串
func (t *translator) value(v string) string {
	if strings.HasPrefix(v, "{$") && strings.HasSuffix(v, "}") && identPathOK(v[2:len(v)-1]) {
		return t.varExpr(v[2 : len(v)-1])
	}
	if strings.HasPrefix(v, "$") && identPathOK(v[1:]) {
		return t.varExpr(v[1:])
	}
	return strconv.Quote(v)
}

func attrs(s string) map[string]string {
	m := map[string]string{}
	for _, a := range attrPattern.FindAllStringSubmatch(s, -1) {
		m[a[1]] = a[2]
	}
	return m
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// identPathOK a 或 a.b.c (变量名, 不含 $)
func identPathOK(s string) bool {
	if s == "" {
		return false
	}
	for _, p := range strings.Split(s, ".") {
		if !identPattern.MatchString(p) {
			return false
		}
	}
	return true
}
