package inertia

import (
	"bufio"
	"encoding/json"
	"net"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	gonertia "github.com/romsar/gonertia/v3"
)

// I is the process-wide Inertia instance, initialized once by Setup.
var I *gonertia.Inertia

const (
	rootTemplatePath = "admin/root.html"
	buildManifest    = "public/build/.vite/manifest.json"
	hotFilePath      = "public/hot"
	entryScript      = "resources/js/app.ts"
	entryStyle       = "resources/css/app.css"
	buildDir         = "/build/"
)

// Setup initializes the Inertia + Vite instance. Must be called once before
// router.SetupRouter().
func Setup() error {
	i, err := gonertia.NewFromFile(rootTemplatePath)
	if err != nil {
		return err
	}

	if _, err := gonertia.NewWithVite(i,
		gonertia.WithBuildManifest(buildManifest),
		gonertia.WithHotFile(hotFilePath),
		gonertia.WithEntryPoints(entryScript, entryStyle),
	); err != nil {
		return err
	}

	// Workaround for a gonertia v3.0.0 bug: viteAssets's processAsset looks
	// up an entry's "css" array entries as manifest KEYS, but Vite's real
	// manifest.json lists those as already-resolved OUTPUT paths (see Vite's
	// backend-integration docs), so the second lookup always misses and the
	// <link> tag is silently dropped. This bites any CSS pulled in
	// transitively through app.ts that isn't itself a declared entry point
	// (e.g. Vue SFC <style scoped> blocks in Login.vue/Sidebar.vue/etc, or a
	// plain `import './x.css'` inside app.ts). Promoting a whole stylesheet
	// to its own rollup entry (as done for app.css/Element Plus) sidesteps
	// the bug for that one file, but SFC-scoped styles have no standalone
	// source file to promote. So in production (non hot-reload) builds we
	// read the manifest ourselves and share the resolved hrefs as template
	// data; root.html renders them as plain <link> tags alongside
	// {{ viteAssets }}. In dev/hot-reload mode this isn't needed: Vite's
	// client injects component styles at runtime via JS.
	if _, statErr := os.Stat(hotFilePath); statErr != nil {
		if hrefs, cssErr := entryCssHrefs(buildManifest, entryScript); cssErr == nil {
			i.ShareTemplateData("viteChunkCss", hrefs)
		}
	}

	I = i
	return nil
}

type viteManifestAsset struct {
	File string   `json:"file"`
	Css  []string `json:"css"`
}

// entryCssHrefs reads the Vite manifest directly and returns build-dir-
// prefixed URLs for the CSS files nested under the given entries' "css"
// arrays (see the Setup doc comment for why this bypasses gonertia's own,
// buggy handling of that field).
func entryCssHrefs(manifestPath string, entries ...string) ([]string, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}

	var manifest map[string]viteManifestAsset
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var hrefs []string
	for _, entry := range entries {
		asset, ok := manifest[entry]
		if !ok {
			continue
		}
		for _, cssPath := range asset.Css {
			if seen[cssPath] {
				continue
			}
			seen[cssPath] = true
			hrefs = append(hrefs, buildDir+cssPath)
		}
	}
	return hrefs, nil
}

// Middleware adapts gonertia's standard-library middleware to Gin.
//
// gonertia's Middleware, for requests carrying the "X-Inertia" header, swaps
// in its own buffering http.ResponseWriter (so it can inspect the handler's
// status/body afterwards and apply Inertia's "empty response -> redirect
// back" and asset-version "full reload" behaviors) and calls the downstream
// handler with THAT writer, not the one it was given. If we ignore the "w"
// argument here and let downstream Gin handlers keep writing to the
// original c.Writer (as a naive adapter would), every write bypasses
// gonertia's buffer: it sees a status-200-empty-body response no matter
// what the real handler did, unconditionally overwrites the just-set
// Location header with its own "go back to referer" redirect, and the
// real body/headers our handler wrote race the connection directly. This
// corrupts real redirects (e.g. POST /login redirecting to /manage/index
// silently becomes a redirect to "/" instead) for any actual Inertia
// client request. So: swap c.Writer to a gin.ResponseWriter view of
// whatever writer gonertia handed us for the duration of the downstream
// call, and restore it after.
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		original := c.Writer
		I.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Request = r
			if grw, ok := w.(gin.ResponseWriter); ok {
				c.Writer = grw
			} else {
				c.Writer = wrapResponseWriter(w)
			}
			c.Next()
		})).ServeHTTP(c.Writer, c.Request)
		c.Writer = original
	}
}

// ginResponseWriter adapts a plain http.ResponseWriter (such as gonertia's
// internal buffering wrapper) to gin's ResponseWriter interface, so Gin
// handlers/middleware can keep using c.Writer transparently.
type ginResponseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func wrapResponseWriter(w http.ResponseWriter) gin.ResponseWriter {
	return &ginResponseWriter{ResponseWriter: w, status: http.StatusOK, size: -1}
}

func (w *ginResponseWriter) WriteHeader(code int) {
	if code > 0 {
		w.status = code
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *ginResponseWriter) WriteHeaderNow() {}

func (w *ginResponseWriter) Write(data []byte) (int, error) {
	n, err := w.ResponseWriter.Write(data)
	if w.size < 0 {
		w.size = 0
	}
	w.size += n
	return n, err
}

func (w *ginResponseWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

func (w *ginResponseWriter) Status() int { return w.status }

func (w *ginResponseWriter) Size() int { return w.size }

func (w *ginResponseWriter) Written() bool { return w.size != -1 }

func (w *ginResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

func (w *ginResponseWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *ginResponseWriter) CloseNotify() <-chan bool {
	if notifier, ok := w.ResponseWriter.(http.CloseNotifier); ok { //nolint:staticcheck
		return notifier.CloseNotify()
	}
	return make(chan bool, 1)
}

func (w *ginResponseWriter) Pusher() http.Pusher {
	if pusher, ok := w.ResponseWriter.(http.Pusher); ok {
		return pusher
	}
	return nil
}
