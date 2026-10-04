package install

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-sql-driver/mysql"
)

func init() { gin.SetMode(gin.TestMode) }

// newTestHandler 在临时目录、无 MySQL / Redis 环境变量的情况下建立安装 handler
func newTestHandler(t *testing.T) *Handler {
	t.Helper()
	t.Chdir(t.TempDir())
	for _, k := range []string{"MYSQL_DSN", "REDIS_ADDR"} {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}
	return NewHandler(&Switch{}, nil)
}

func validForm(code string) url.Values {
	return url.Values{
		"mysql_host": {"myhost"}, "mysql_port": {"3306"}, "mysql_user": {"root"},
		"mysql_password": {"secretpw"}, "mysql_db": {"FilmSite"},
		"redis_addr": {"127.0.0.1:6379"}, "redis_password": {"redispw"}, "redis_db": {"0"},
		"account": {"boss"}, "email": {"boss@example.com"},
		"password": {"adminpw1"}, "password2": {"adminpw1"}, "code": {code},
	}
}

func post(h http.Handler, path string, form url.Values) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestFormDSN(t *testing.T) {
	f := Form{MysqlHost: "db.local", MysqlPort: "3307", MysqlUser: "u", MysqlPassword: "p@ss:w/rd?&#", MysqlDB: "FilmSite"}
	cfg, err := mysql.ParseDSN(f.DSN())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Passwd != "p@ss:w/rd?&#" || cfg.User != "u" || cfg.Addr != "db.local:3307" || cfg.DBName != "FilmSite" {
		t.Errorf("round trip = %+v", cfg)
	}
	if !cfg.ParseTime || cfg.Loc.String() != "Local" || !strings.Contains(f.DSN(), "charset=utf8mb4") {
		t.Errorf("DSN options missing: %s", f.DSN())
	}
}

func TestRedirectsToInstall(t *testing.T) {
	h := newTestHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/index", nil))
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/install" {
		t.Errorf("got %d %q", rec.Code, rec.Header().Get("Location"))
	}
}

func TestInstallPageShowsChecksAndForm(t *testing.T) {
	h := newTestHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/install", nil))
	body := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.Contains(body, "环境检查") || !strings.Contains(body, `name="mysql_host"`) {
		t.Errorf("page = %d\n%s", rec.Code, body)
	}
	// 临时目录不是项目根目录 → 有阻挡项, 安装按钮禁用
	if !strings.Contains(body, "工作目录") || !strings.Contains(body, `<button type="submit" disabled>`) {
		t.Error("blocking check should disable the install button")
	}
}

func TestWrongCodeRejected(t *testing.T) {
	h := newTestHandler(t)
	rec := post(h, "/install", validForm("000000x"))
	if !strings.Contains(rec.Body.String(), "安装码错误") {
		t.Errorf("body should report wrong code:\n%s", rec.Body.String())
	}
	if Locked() {
		t.Error("lock must not be written")
	}
	if rec := post(h, "/install/test", validForm("000000x")); !strings.Contains(rec.Body.String(), `"code":-1`) {
		t.Errorf("test endpoint should reject wrong code: %s", rec.Body.String())
	}
}

func TestCodeIsTrimmed(t *testing.T) {
	h := newTestHandler(t)
	form := validForm("  " + h.code + "\n")
	form.Set("account", "ab") // 账号无效 → 停在表单校验, 不连数据库
	body := post(h, "/install", form).Body.String()
	if strings.Contains(body, "安装码错误") || !strings.Contains(body, "管理员账号") {
		t.Errorf("trimmed code should pass and fail on account:\n%s", body)
	}
}

func TestValidationKeepsValuesClearsPasswords(t *testing.T) {
	h := newTestHandler(t)
	form := validForm(h.code)
	form.Set("password2", "different")
	body := post(h, "/install", form).Body.String()
	if !strings.Contains(body, "两次输入的密码不一致") {
		t.Errorf("missing validation error:\n%s", body)
	}
	if !strings.Contains(body, `value="myhost"`) || !strings.Contains(body, `value="boss@example.com"`) {
		t.Error("non-password values should be kept")
	}
	for _, secret := range []string{"secretpw", "redispw", "adminpw1"} {
		if strings.Contains(body, secret) {
			t.Errorf("password %q must not be echoed back", secret)
		}
	}
}

func TestEnvProvidedGroupNotEditable(t *testing.T) {
	h := newTestHandler(t)
	t.Setenv("MYSQL_DSN", "root:x@tcp(127.0.0.1:1)/FilmSite")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/install", nil))
	body := rec.Body.String()
	if !strings.Contains(body, "由环境变量提供") || strings.Contains(body, `name="mysql_host"`) {
		t.Errorf("MySQL group should be read-only:\n%s", body)
	}
	if !strings.Contains(body, `name="redis_addr"`) {
		t.Error("Redis group should still be editable")
	}
}

func TestSubmitWhileInstalling(t *testing.T) {
	h := newTestHandler(t)
	h.mu.Lock()
	defer h.mu.Unlock()
	rec := post(h, "/install", validForm(h.code))
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "安装正在进行中") {
		t.Errorf("got %d\n%s", rec.Code, rec.Body.String())
	}
}
