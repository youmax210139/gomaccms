package renderer

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"gomaccms/internal/view/maccms"

	"github.com/gin-gonic/gin"
)

// ThemesRoot is the directory holding every theme. Each theme is one
// self-contained directory:
//
//	themes/<name>/templates/layouts/*.html    server-side layout(s), define "layout"
//	themes/<name>/templates/components/*.html shared partials (header, footer, ...)
//	themes/<name>/templates/pages/*.html      one file per page, each defines "content"
//	themes/<name>/public/                     browser assets, served at /static/<name>/
//	themes/<name>/lang/<code>.json            界面文字语言包 (见 i18n.go)
//
// Only public/ is exposed over HTTP; templates/ is never served.
const ThemesRoot = "themes"

// ThemeBundle holds one theme's parsed page templates and its static asset URL base.
type ThemeBundle struct {
	Pages     map[string]*template.Template
	AssetPath string
	// Texts 语言包: 语言代码 → 文字 (已合并原文语言包)
	Texts map[string]map[string]string
}

// Registry maps theme name -> its ThemeBundle. Populated once at boot by LoadThemes.
var Registry = map[string]*ThemeBundle{}

// reload makes Render re-parse the requested theme's templates on every
// request, so template edits show up without restarting the server. It is
// enabled whenever Gin is not in release mode (i.e. local development);
// production keeps the templates parsed once at boot.
var reload bool

// LoadThemes walks themes/*/, parses every theme's pages, and registers each
// theme's public/ directory as a static route. A theme that fails to parse
// (including MacCMS tag errors) is logged and skipped instead of panicking.
func LoadThemes(r *gin.Engine) {
	reload = gin.Mode() != gin.ReleaseMode
	entries, err := os.ReadDir(ThemesRoot)
	if err != nil {
		log.Panicf("theme: failed to read themes root %q: %v", ThemesRoot, err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		bundle, err := parseTheme(name)
		if err != nil {
			log.Println("theme: skipped:", err)
			continue
		}
		r.Static(bundle.AssetPath, filepath.Join(ThemesRoot, name, "public"))
		Registry[name] = bundle
	}
}

// parseTheme parses one theme's pages. Each page gets its own
// *template.Template (layouts + components + that page), so every page's
// {{define "content"}} block can't collide with another page's.
func parseTheme(name string) (*ThemeBundle, error) {
	tplDir := filepath.Join(ThemesRoot, name, "templates")

	layoutPaths, err := filepath.Glob(filepath.Join(tplDir, "layouts", "*.html"))
	if err != nil {
		return nil, fmt.Errorf("theme %s: failed to glob layouts: %w", name, err)
	}
	componentPaths, err := filepath.Glob(filepath.Join(tplDir, "components", "*.html"))
	if err != nil {
		return nil, fmt.Errorf("theme %s: failed to glob components: %w", name, err)
	}
	pagePaths, err := filepath.Glob(filepath.Join(tplDir, "pages", "*.html"))
	if err != nil {
		return nil, fmt.Errorf("theme %s: failed to glob pages: %w", name, err)
	}

	shared := append(layoutPaths, componentPaths...)
	pages := map[string]*template.Template{}
	for _, pagePath := range pagePaths {
		base := filepath.Base(pagePath)
		pageName := strings.TrimSuffix(base, filepath.Ext(base))

		parseFiles := append(append([]string{}, shared...), pagePath)
		tmpl, err := parseThemeFiles(tplDir, base, parseFiles)
		if err != nil {
			return nil, fmt.Errorf("theme %s: failed to parse page %q: %w", name, pageName, err)
		}
		pages[pageName] = tmpl
	}

	texts, err := loadTexts(filepath.Join(ThemesRoot, name, "lang"))
	if err != nil {
		return nil, fmt.Errorf("theme %s: %w", name, err)
	}
	return &ThemeBundle{Pages: pages, AssetPath: fmt.Sprintf("/static/%s", name), Texts: texts}, nil
}

// ThemeNames returns the names of every usable theme, sorted. A theme is
// usable when it has at least one page and that page's template set defines
// "layout" (the entry point Render executes); bare directories under
// themes/ without that structure are left out.
func ThemeNames() []string {
	var names []string
	for name, bundle := range Registry {
		for _, tmpl := range bundle.Pages {
			if tmpl.Lookup("layout") != nil {
				names = append(names, name)
			}
			break
		}
	}
	sort.Strings(names)
	return names
}

// HasTheme reports whether name is one of ThemeNames.
func HasTheme(name string) bool {
	return slices.Contains(ThemeNames(), name)
}

// parseThemeFiles 与 template.ParseFiles 相同 (每个文件一个以文件名命名的模板, name 为第一个),
// 但先把文件中的 MacCMS 标签翻译成 Go 模板语法 ({include file="x"} 读取 tplDir/x.html)
func parseThemeFiles(tplDir, name string, paths []string) (*template.Template, error) {
	include := func(file string) (string, error) {
		b, err := os.ReadFile(filepath.Join(tplDir, file+".html"))
		return string(b), err
	}
	root := template.New(name).Funcs(FuncMap)
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		src, err := maccms.Translate(string(b), include)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", filepath.Base(p), err)
		}
		t := root
		if base := filepath.Base(p); base != name {
			t = root.New(base)
		}
		if _, err := t.Parse(src); err != nil {
			return nil, err
		}
	}
	return root, nil
}
