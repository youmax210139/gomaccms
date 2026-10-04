package renderer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

func writePack(t *testing.T, dir, lang, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, lang+".json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadTextsMergesOverSource(t *testing.T) {
	dir := t.TempDir()
	writePack(t, dir, "zh-CN", `{"a":"甲","b":"乙"}`)
	writePack(t, dir, "vi", `{"a":"A"}`)
	texts, err := loadTexts(dir)
	if err != nil {
		t.Fatal(err)
	}
	if texts["vi"]["a"] != "A" || texts["vi"]["b"] != "乙" {
		t.Fatalf("vi must override a and fall back to zh-CN for b: %v", texts["vi"])
	}
	if texts["zh-CN"]["a"] != "甲" {
		t.Fatalf("zh-CN pack wrong: %v", texts["zh-CN"])
	}
}

func TestLoadTextsMissingDirAndBadJSON(t *testing.T) {
	texts, err := loadTexts(filepath.Join(t.TempDir(), "nope"))
	if err != nil || len(texts) != 0 {
		t.Fatalf("missing dir must give empty packs: %v %v", texts, err)
	}
	dir := t.TempDir()
	writePack(t, dir, "zh-CN", `{bad`)
	if _, err := loadTexts(dir); err == nil {
		t.Fatal("bad JSON must fail")
	}
}

func TestBundleTextsFallsBackToSource(t *testing.T) {
	b := &ThemeBundle{Texts: map[string]map[string]string{"zh-CN": {"k": "中"}}}
	if b.texts("fr")["k"] != "中" || b.texts("")["k"] != "中" {
		t.Fatal("unknown or empty language must use the zh-CN pack")
	}
}

func TestTranslate(t *testing.T) {
	data := map[string]any{"i18n": map[string]string{"hi": "你好 %s", "plain": "纯文本"}}
	if got := translate(data, "hi", "小明"); got != "你好 小明" {
		t.Fatalf("args: %q", got)
	}
	if got := translate(data, "plain"); got != "纯文本" {
		t.Fatalf("no args: %q", got)
	}
	if got := translate(data, "missing.key"); got != "missing.key" {
		t.Fatalf("missing key must return the key: %q", got)
	}
	if got := translate(map[string]any{}, "x"); got != "x" {
		t.Fatalf("no pack must return the key: %q", got)
	}
}

func TestTranslateHTMLEscapesArgs(t *testing.T) {
	data := map[string]any{"i18n": map[string]string{"r": `搜索 "<strong>%s</strong>" 共 <strong>%d</strong>`}}
	got := string(translateHTML(data, "r", `<script>x</script>`, 3))
	want := `搜索 "<strong>&lt;script&gt;x&lt;/script&gt;</strong>" 共 <strong>3</strong>`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

var verbPattern = regexp.MustCompile(`%[sdv]`)

// TestDefaultThemePacksComplete 默认主题的每个语言包都要包含原文语言包的全部 key, 且格式参数的种类与顺序一致
func TestDefaultThemePacksComplete(t *testing.T) {
	dir := filepath.Join("..", "..", "..", "themes", "default", "lang")
	read := func(lang string) map[string]string {
		b, err := os.ReadFile(filepath.Join(dir, lang+".json"))
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]string
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatalf("%s: %v", lang, err)
		}
		return m
	}
	src := read("zh-CN")
	files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	if len(files) < 2 {
		t.Fatalf("expected zh-CN.json and vi.json in %s", dir)
	}
	for _, f := range files {
		lang := strings.TrimSuffix(filepath.Base(f), ".json")
		for _, p := range packProblems(src, read(lang)) {
			t.Errorf("%s: %s", lang, p)
		}
	}
}

// packProblems 语言包与原文语言包的差异: 缺少 / 多出的 key, 以及格式参数不一致
func packProblems(src, pack map[string]string) []string {
	var out []string
	for k, v := range src {
		tv, ok := pack[k]
		if !ok {
			out = append(out, fmt.Sprintf("missing key %q", k))
			continue
		}
		if !slices.Equal(verbPattern.FindAllString(v, -1), verbPattern.FindAllString(tv, -1)) {
			out = append(out, fmt.Sprintf("key %q has different format verbs: %q vs %q", k, v, tv))
		}
	}
	for k := range pack {
		if _, ok := src[k]; !ok {
			out = append(out, fmt.Sprintf("key %q not in zh-CN.json", k))
		}
	}
	return out
}

// TestPackProblemsVerbOrder 参数顺序不同 (如 %s 与 %d 对调) 时 fmt 会输出乱码, 必须检查出来
func TestPackProblemsVerbOrder(t *testing.T) {
	src := map[string]string{"k": "搜索 %s 找到 %d 部"}
	if p := packProblems(src, map[string]string{"k": "có %d phim cho %s"}); len(p) == 0 {
		t.Fatal("reordered verbs must be reported")
	}
	if p := packProblems(src, map[string]string{"k": "tìm %s có %d phim"}); len(p) != 0 {
		t.Fatalf("same order must pass: %v", p)
	}
}
