package maccms

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// condition {if condition="..."} 的条件翻译成 Go 模板的管线. 只支持:
// 变量 ($a.b)、数字、'字串' / "字串"; 比较 == != > < >= <= (或 eq neq gt lt egt elt);
// && || ! (或 and or not); 括号. 单独的变量按 MacCMS 的空值判断真假
func (t *translator) condition(body string) (string, error) {
	cond, ok := attrs(body)["condition"]
	if !ok || strings.TrimSpace(cond) == "" {
		return "", fmt.Errorf("缺少 condition")
	}
	toks, err := lex(cond)
	if err != nil {
		return "", err
	}
	p := &condParser{t: t, toks: toks}
	out, err := p.or()
	if err != nil {
		return "", err
	}
	if p.pos < len(p.toks) {
		return "", fmt.Errorf("条件中有无法识别的 %q", p.toks[p.pos].text)
	}
	return out, nil
}

type tokKind int

const (
	tokVar tokKind = iota
	tokNum
	tokStr
	tokCmp
	tokAnd
	tokOr
	tokNot
	tokLParen
	tokRParen
)

type token struct {
	kind tokKind
	text string
}

var lexPattern = regexp.MustCompile(`^(?:(\$[A-Za-z_][\w.]*)|(-?\d+(?:\.\d+)?)|'([^']*)'|"([^"]*)"|(===|!==|==|!=|>=|<=|>|<)|(&&|\|\|)|(!)|(\()|(\))|([A-Za-z]+))`)

// 文字形式的运算符
var wordOps = map[string]token{
	"eq": {tokCmp, "eq"}, "neq": {tokCmp, "neq"}, "heq": {tokCmp, "eq"}, "nheq": {tokCmp, "neq"},
	"gt": {tokCmp, "gt"}, "lt": {tokCmp, "lt"}, "egt": {tokCmp, "egt"}, "elt": {tokCmp, "elt"},
	"and": {tokAnd, "and"}, "or": {tokOr, "or"}, "not": {tokNot, "not"},
}

var symbolOps = map[string]string{"==": "eq", "===": "eq", "!=": "neq", "!==": "neq", ">": "gt", "<": "lt", ">=": "egt", "<=": "elt"}

func lex(s string) ([]token, error) {
	var toks []token
	for s = strings.TrimSpace(s); s != ""; s = strings.TrimSpace(s) {
		m := lexPattern.FindStringSubmatch(s)
		if m == nil {
			return nil, fmt.Errorf("条件中有无法识别的 %q", s)
		}
		switch {
		case m[1] != "":
			toks = append(toks, token{tokVar, m[1][1:]})
		case m[2] != "":
			toks = append(toks, token{tokNum, m[2]})
		case strings.HasPrefix(m[0], "'"):
			toks = append(toks, token{tokStr, m[3]})
		case strings.HasPrefix(m[0], `"`):
			toks = append(toks, token{tokStr, m[4]})
		case m[5] != "":
			toks = append(toks, token{tokCmp, symbolOps[m[5]]})
		case m[6] == "&&":
			toks = append(toks, token{tokAnd, "and"})
		case m[6] == "||":
			toks = append(toks, token{tokOr, "or"})
		case m[7] != "":
			toks = append(toks, token{tokNot, "not"})
		case m[8] != "":
			toks = append(toks, token{tokLParen, "("})
		case m[9] != "":
			toks = append(toks, token{tokRParen, ")"})
		default:
			op, ok := wordOps[strings.ToLower(m[10])]
			if !ok {
				return nil, fmt.Errorf("条件中不支持 %q", m[10])
			}
			toks = append(toks, op)
		}
		s = s[len(m[0]):]
	}
	return toks, nil
}

type condParser struct {
	t    *translator
	toks []token
	pos  int
}

func (p *condParser) peek(kind tokKind) bool {
	return p.pos < len(p.toks) && p.toks[p.pos].kind == kind
}

// or := and (|| and)*
func (p *condParser) or() (string, error) {
	return p.chain(tokOr, "or", p.and)
}

// and := unary (&& unary)*
func (p *condParser) and() (string, error) {
	return p.chain(tokAnd, "and", p.unary)
}

func (p *condParser) chain(kind tokKind, fn string, next func() (string, error)) (string, error) {
	left, err := next()
	if err != nil {
		return "", err
	}
	for p.peek(kind) {
		p.pos++
		right, err := next()
		if err != nil {
			return "", err
		}
		left = "(" + fn + " " + left + " " + right + ")"
	}
	return left, nil
}

// unary := ! unary | compare
func (p *condParser) unary() (string, error) {
	if p.peek(tokNot) {
		p.pos++
		x, err := p.unary()
		if err != nil {
			return "", err
		}
		return "(not " + x + ")", nil
	}
	return p.compare()
}

// compare := operand (op operand)?
func (p *condParser) compare() (string, error) {
	if p.peek(tokLParen) {
		p.pos++
		x, err := p.or()
		if err != nil {
			return "", err
		}
		if !p.peek(tokRParen) {
			return "", fmt.Errorf("条件缺少 )")
		}
		p.pos++
		return x, nil
	}
	left, err := p.operand()
	if err != nil {
		return "", err
	}
	if !p.peek(tokCmp) {
		return "(not (maccmsEmpty " + left + "))", nil
	}
	op := p.toks[p.pos].text
	p.pos++
	right, err := p.operand()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("(maccmsCmp %q %s %s)", op, left, right), nil
}

func (p *condParser) operand() (string, error) {
	if p.pos >= len(p.toks) {
		return "", fmt.Errorf("条件不完整")
	}
	tk := p.toks[p.pos]
	p.pos++
	switch tk.kind {
	case tokVar:
		if !identPathOK(tk.text) {
			return "", fmt.Errorf("变量 %q 不合法", tk.text)
		}
		return p.t.varExpr(tk.text), nil
	case tokNum, tokStr:
		return strconv.Quote(tk.text), nil
	}
	return "", fmt.Errorf("条件中 %q 的位置不对", tk.text)
}
