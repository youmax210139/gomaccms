package renderer

import "testing"

// 简介的空白: 中文之间的空白 (含全角缩进) 去掉, 拉丁文字的词间空格保留为一个
func TestCleanFilmContentSpaces(t *testing.T) {
	cases := map[string]string{
		"<p>　　一名男子 旅行归来。</p>\n":       "一名男子旅行归来。",
		"<p>Tóm tắt  thử\nnghiệm</p>": "Tóm tắt thử nghiệm",
		"讲述 Tony Stark 的故事":           "讲述Tony Stark的故事",
	}
	for in, want := range cases {
		if got := cleanFilmContent(in); got != want {
			t.Errorf("cleanFilmContent(%q) = %q, want %q", in, got, want)
		}
	}
	if got := searchBlurb("<p>Một  người\tđàn ông</p>"); got != "Một người đàn ông" {
		t.Errorf("searchBlurb = %q", got)
	}
}
