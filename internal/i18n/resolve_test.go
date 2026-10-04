package i18n

import (
	"reflect"
	"testing"
)

func enabledSet(codes ...string) func(string) bool {
	m := map[string]bool{}
	for _, c := range codes {
		m[c] = true
	}
	return func(code string) bool { return m[code] }
}

func TestResolve(t *testing.T) {
	cases := []struct {
		name, cookie, def string
		langs             []string
		enabled           []string
		want              string
	}{
		{"cookie allowed", "vi", "zh-CN", []string{"zh-CN", "vi"}, []string{"zh-CN", "vi"}, "vi"},
		{"no cookie uses default", "", "vi", []string{"zh-CN", "vi"}, []string{"zh-CN", "vi"}, "vi"},
		{"cookie not in scheme", "en", "zh-CN", []string{"zh-CN", "vi"}, []string{"zh-CN", "vi", "en"}, "zh-CN"},
		{"cookie disabled", "vi", "zh-CN", []string{"zh-CN", "vi"}, []string{"zh-CN"}, "zh-CN"},
		{"default disabled falls to first enabled scheme lang", "", "vi", []string{"vi", "en"}, []string{"zh-CN", "en"}, "en"},
		{"nothing enabled falls to source", "", "vi", []string{"vi"}, []string{"zh-CN"}, SourceLang},
		{"empty scheme config", "", "", nil, []string{"zh-CN"}, SourceLang},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Resolve(c.cookie, c.def, c.langs, enabledSet(c.enabled...)); got != c.want {
				t.Fatalf("Resolve() = %q, want %q", got, c.want)
			}
		})
	}
}

func TestAvailable(t *testing.T) {
	all := []Language{
		{Code: "zh-CN", Name: "中文", Enabled: true},
		{Code: "vi", Name: "Tiếng Việt", Enabled: true},
		{Code: "en", Name: "English", Enabled: false},
	}
	codes := func(l []Language) []string {
		out := []string{}
		for _, x := range l {
			out = append(out, x.Code)
		}
		return out
	}
	if got := codes(Available(all, []string{"vi", "zh-CN", "en"})); !reflect.DeepEqual(got, []string{"zh-CN", "vi"}) {
		t.Fatalf("Available keeps language order and drops disabled: got %v", got)
	}
	if got := codes(Available(all, []string{"en"})); !reflect.DeepEqual(got, []string{"zh-CN"}) {
		t.Fatalf("Available falls back to source language: got %v", got)
	}
}

func TestCheckSchemeLangs(t *testing.T) {
	en := enabledSet("zh-CN", "vi")
	def, langs, err := CheckSchemeLangs("", []string{" vi ", "zh-CN", "vi", ""}, en)
	if err != nil || def != "vi" || !reflect.DeepEqual(langs, []string{"vi", "zh-CN"}) {
		t.Fatalf("trim/dedupe/default-to-first: got %q %v %v", def, langs, err)
	}
	if _, _, err := CheckSchemeLangs("zh-CN", nil, en); err == nil {
		t.Fatal("empty langs must fail")
	}
	if _, _, err := CheckSchemeLangs("zh-CN", []string{"zh-CN", "en"}, en); err == nil {
		t.Fatal("disabled language must fail")
	}
	if _, _, err := CheckSchemeLangs("vi", []string{"zh-CN"}, en); err == nil {
		t.Fatal("default outside langs must fail")
	}
}

func TestValidateLanguage(t *testing.T) {
	ok := []Language{{Code: "vi", Name: "Tiếng Việt", LibreCode: "vi"}, {Code: "zh-TW", Name: "繁體", LibreCode: "zt"}, {Code: "pt-BR", Name: "Português"}}
	for _, l := range ok {
		l := l
		if err := ValidateLanguage(&l); err != nil {
			t.Fatalf("%s should be valid: %v", l.Code, err)
		}
	}
	bad := []Language{{Code: "", Name: "x"}, {Code: "VI", Name: "x"}, {Code: "vi_VN", Name: "x"}, {Code: "vi", Name: ""}, {Code: "vi", Name: "x", LibreCode: "v i"}, {Code: "vi", Name: "x", Sort: -1}}
	for _, l := range bad {
		l := l
		if err := ValidateLanguage(&l); err == nil {
			t.Fatalf("%+v should be invalid", l)
		}
	}
	l := Language{Code: " vi ", Name: " Tiếng Việt ", LibreCode: " vi "}
	if err := ValidateLanguage(&l); err != nil || l.Code != "vi" || l.Name != "Tiếng Việt" || l.LibreCode != "vi" {
		t.Fatalf("ValidateLanguage must trim fields: %+v %v", l, err)
	}
}
