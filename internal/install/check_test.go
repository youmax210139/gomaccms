package install

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gomaccms/internal/config"
)

func TestVersionAtLeast(t *testing.T) {
	cases := []struct {
		version, min string
		want         bool
	}{
		{"8.4.2", "5.7.8", true},
		{"5.7.44-log", "5.7.8", true},
		{"5.7.7", "5.7.8", false},
		{"10.11.6-MariaDB", "5.7.8", true},
		{"7.2.5", "5.0.0", true},
		{"4.0.9", "5.0.0", false},
		{"5.0", "5.0.0", true},
		{"garbage", "5.0.0", false},
	}
	for _, c := range cases {
		if got := versionAtLeast(c.version, c.min); got != c.want {
			t.Errorf("versionAtLeast(%q, %q) = %v, want %v", c.version, c.min, got, c.want)
		}
	}
}

func TestInfoField(t *testing.T) {
	info := "# Server\r\nredis_version:7.2.5\r\nredis_mode:standalone\r\n"
	if got := infoField(info, "redis_version"); got != "7.2.5" {
		t.Errorf("infoField = %q", got)
	}
	if got := infoField(info, "nope"); got != "" {
		t.Errorf("infoField missing = %q", got)
	}
}

// projectLayout 在临时目录建出运行所需的文件
func projectLayout(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	for _, p := range []string{"themes/default/templates", "public/build/.vite"} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range []string{"public/build/.vite/manifest.json", "admin/root.html"} {
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func findCheck(checks []Check, name string) Check {
	for _, c := range checks {
		if c.Name == name {
			return c
		}
	}
	return Check{}
}

func TestCheckEnvironmentOK(t *testing.T) {
	projectLayout(t)
	checks := CheckEnvironment()
	if msg := Blocked(checks); msg != "" {
		t.Fatalf("Blocked = %q, want none", msg)
	}
	if _, err := os.Stat("storage/upload/gallery"); err != nil {
		t.Error("storage/upload/gallery should have been created")
	}
}

func TestCheckEnvironmentMissingFiles(t *testing.T) {
	t.Chdir(t.TempDir())
	c := findCheck(CheckEnvironment(), "工作目录")
	if c.OK || !c.Blocking {
		t.Fatalf("工作目录 check = %+v, want blocking failure", c)
	}
	if !strings.Contains(c.Detail, "项目根目录") || !strings.Contains(c.Detail, "admin/root.html") {
		t.Errorf("Detail should name missing files and the fix, got %q", c.Detail)
	}
}

func TestCheckEnvironmentReadOnlyStorage(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores permissions")
	}
	projectLayout(t)
	if err := os.Mkdir("storage", 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod("storage", 0o755) })
	c := findCheck(CheckEnvironment(), "目录可写")
	if c.OK || !c.Blocking {
		t.Fatalf("目录可写 check = %+v, want blocking failure", c)
	}
	if !strings.Contains(c.Detail, "chmod") {
		t.Errorf("Detail should suggest chmod, got %q", c.Detail)
	}
}

func TestCheckMysqlUnreachable(t *testing.T) {
	c := CheckMysql("root:x@tcp(127.0.0.1:1)/FilmSite")
	if c.OK || !c.Blocking || !strings.Contains(c.Detail, "连接失败") {
		t.Errorf("CheckMysql = %+v", c)
	}
}

func TestCheckRedisUnreachable(t *testing.T) {
	c := CheckRedis("127.0.0.1:1", "", 0)
	if c.OK || !c.Blocking || !strings.Contains(c.Detail, "连接失败") {
		t.Errorf("CheckRedis = %+v", c)
	}
}

func TestDetectReportsUnreachableDatabase(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("MYSQL_DSN", "root:x@tcp(127.0.0.1:1)/FilmSite")
	config.Load()
	t.Cleanup(config.Load)
	installed, err := Detect()
	if installed || err == nil {
		t.Fatalf("Detect = %v, %v; want false with a connection error", installed, err)
	}
	if !Configured() {
		t.Error("MYSQL_DSN from env means the database is configured")
	}
	if Locked() {
		t.Error("no lock on connection failure")
	}
}

func TestConfiguredFalseOnFreshMachine(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("MYSQL_DSN", "")
	os.Unsetenv("MYSQL_DSN")
	if Configured() {
		t.Error("no env and no config.env: not configured")
	}
	os.MkdirAll("storage", 0o755)
	os.WriteFile(config.EnvFilePath, []byte("REDIS_DB=1\n"), 0o600)
	if !Configured() {
		t.Error("an existing config.env means a previous install attempt configured the database")
	}
}

func TestLock(t *testing.T) {
	t.Chdir(t.TempDir())
	if Locked() {
		t.Fatal("no lock yet")
	}
	if err := WriteLock(); err != nil {
		t.Fatal(err)
	}
	if !Locked() {
		t.Fatal("lock should exist")
	}
}

func TestSwitch(t *testing.T) {
	var s Switch
	s.Set(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("a")) }))
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	s.Set(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("b")) }))
	rec2 := httptest.NewRecorder()
	s.ServeHTTP(rec2, httptest.NewRequest("GET", "/", nil))
	if rec.Body.String() != "a" || rec2.Body.String() != "b" {
		t.Errorf("got %q then %q", rec.Body.String(), rec2.Body.String())
	}
}
