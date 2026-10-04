package i18n

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"
)

var (
	codePattern  = regexp.MustCompile(`^[a-z]{2,3}(-[A-Za-z0-9]{2,8})?$`)
	librePattern = regexp.MustCompile(`^[A-Za-z-]*$`)
)

// Resolve 请求使用的语言: cookie 是方案启用且全局启用的语言时用它, 否则方案默认语言;
// 默认语言不可用时取方案中第一个仍启用的语言, 都不可用时为原文语言
func Resolve(cookie, schemeDefault string, schemeLangs []string, enabled func(string) bool) string {
	allowed := func(code string) bool { return code != "" && slices.Contains(schemeLangs, code) && enabled(code) }
	switch {
	case allowed(cookie):
		return cookie
	case allowed(schemeDefault):
		return schemeDefault
	}
	for _, code := range schemeLangs {
		if enabled(code) {
			return code
		}
	}
	return SourceLang
}

// Available 前台语言切换可选的语言: 方案启用且全局启用的语言 (按语言列表的顺序); 没有时只有原文语言
func Available(all []Language, schemeLangs []string) []Language {
	out := []Language{}
	for _, l := range all {
		if l.Enabled && slices.Contains(schemeLangs, l.Code) {
			out = append(out, l)
		}
	}
	if len(out) == 0 {
		for _, l := range all {
			if l.Code == SourceLang {
				out = append(out, l)
			}
		}
	}
	return out
}

// CheckSchemeLangs 校验并整理分类方案的语言设置: 去除空白与重复, 至少一种, 都必须是启用的语言;
// 默认语言为空时取第一个, 且必须在启用的语言中
func CheckSchemeLangs(def string, langs []string, enabled func(string) bool) (string, []string, error) {
	var out []string
	for _, code := range langs {
		code = strings.TrimSpace(code)
		if code == "" || slices.Contains(out, code) {
			continue
		}
		if !enabled(code) {
			return "", nil, fmt.Errorf("语言 %s 不存在或已停用", code)
		}
		out = append(out, code)
	}
	if len(out) == 0 {
		return "", nil, errors.New("至少选择一种语言")
	}
	def = strings.TrimSpace(def)
	if def == "" {
		def = out[0]
	}
	if !slices.Contains(out, def) {
		return "", nil, errors.New("默认语言必须是启用的语言之一")
	}
	return def, out, nil
}

// ValidateLanguage 校验语言表单 (去除首尾空白)
func ValidateLanguage(l *Language) error {
	l.Code, l.Name, l.LibreCode = strings.TrimSpace(l.Code), strings.TrimSpace(l.Name), strings.TrimSpace(l.LibreCode)
	switch {
	case !codePattern.MatchString(l.Code):
		return errors.New("语言代码格式错误, 如 vi、en、zh-TW")
	case l.Name == "" || utf8.RuneCountInString(l.Name) > 40:
		return errors.New("名称不能为空, 且不能超过 40 个字符")
	case len(l.LibreCode) > 16 || !librePattern.MatchString(l.LibreCode):
		return errors.New("翻译服务代码只能包含字母与 -, 且不能超过 16 个字符")
	case l.Sort < 0:
		return errors.New("排序值不能为负数")
	}
	return nil
}
