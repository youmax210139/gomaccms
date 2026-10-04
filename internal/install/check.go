package install

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"gomaccms/internal/config"

	"github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
)

// Check 一项环境检查的结果
type Check struct {
	Name     string `json:"name"`
	OK       bool   `json:"ok"`
	Detail   string `json:"detail"`   // 版本号 / 失败原因 / 修正提示
	Blocking bool   `json:"blocking"` // 不通过时阻止安装
}

const (
	minMysql = "5.7.8" // schema 使用 JSON 列与 ngram 全文索引 (MariaDB 没有 ngram, 不支持)
	minRedis = "5.0.0"
)

// Blocked 第一个未通过的阻挡项 ("名称: 说明"), 全部通过时返回 ""
func Blocked(checks []Check) string {
	for _, c := range checks {
		if c.Blocking && !c.OK {
			return c.Name + ": " + c.Detail
		}
	}
	return ""
}

// CheckEnvironment 运行环境 (仅展示)、工作目录、storage 可写
func CheckEnvironment() []Check {
	checks := []Check{{
		Name:   "运行环境",
		OK:     true,
		Detail: fmt.Sprintf("%s %s/%s", runtime.Version(), runtime.GOOS, runtime.GOARCH),
	}}

	var missing []string
	for _, p := range []string{"themes/default/templates", "admin/root.html"} {
		if !exists(p) {
			missing = append(missing, p)
		}
	}
	if !exists("public/build/.vite/manifest.json") && !exists("public/hot") {
		missing = append(missing, "public/build (请先执行 cd admin && npm install && npm run build)")
	}
	wd, _ := os.Getwd()
	dirCheck := Check{Name: "工作目录", OK: len(missing) == 0, Detail: wd, Blocking: true}
	if !dirCheck.OK {
		dirCheck.Detail = "缺少 " + strings.Join(missing, ", ") + "; 请在项目根目录 (含 themes/、public/、admin/) 运行, 当前为 " + wd
	}

	writeCheck := Check{Name: "目录可写", OK: true, Detail: "storage/", Blocking: true}
	for _, dir := range []string{"storage", config.FilmPictureUploadDir, config.SitemapDir} {
		if err := writable(dir); err != nil {
			writeCheck.OK = false
			writeCheck.Detail = dir + " 不可写 (" + err.Error() + "); 请用 chmod / chown 让运行用户可写该目录"
			break
		}
	}
	return append(checks, dirCheck, writeCheck)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// writable 建立目录并写入、删除一个测试文件
func writable(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".install-check-*")
	if err != nil {
		return err
	}
	f.Close()
	return os.Remove(f.Name())
}

// CheckMysql 连接 MySQL 服务器 (不指定库, 库不存在时由安装步骤创建) 并检查版本
func CheckMysql(dsn string) Check {
	c := Check{Name: "MySQL", Blocking: true}
	version, err := mysqlVersion(dsn)
	if err != nil {
		c.Detail = "连接失败: " + err.Error()
		return c
	}
	return mysqlVersionCheck(version)
}

// mysqlVersionCheck 按 SELECT VERSION() 的结果判断服务器是否可用
func mysqlVersionCheck(version string) Check {
	c := Check{Name: "MySQL", Blocking: true}
	if strings.Contains(strings.ToLower(version), "mariadb") {
		c.Detail = version + " (不支持 MariaDB: 缺少影片搜索所需的 ngram 全文解析器, 请改用 MySQL " + minMysql + " 以上, 建议 8.0)"
		return c
	}
	if !versionAtLeast(version, minMysql) {
		c.Detail = version + " (需要 " + minMysql + " 以上)"
		return c
	}
	c.OK, c.Detail = true, version
	if !versionAtLeast(version, "8.0.0") {
		c.Detail += " (建议升级到 8.0 以上)"
	}
	return c
}

func mysqlVersion(dsn string) (string, error) {
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return "", err
	}
	cfg.DBName = ""
	cfg.Timeout = 5 * time.Second
	conn, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return "", err
	}
	defer conn.Close()
	var version string
	err = conn.QueryRow("SELECT VERSION()").Scan(&version)
	return version, err
}

// CheckRedis 连接 Redis (含密码与库号) 并检查版本
func CheckRedis(addr, password string, dbNo int) Check {
	c := Check{Name: "Redis", Blocking: true}
	rdb := redis.NewClient(&redis.Options{Addr: addr, Password: password, DB: dbNo, DialTimeout: 5 * time.Second, MaxRetries: -1})
	defer rdb.Close()
	info, err := rdb.Info(context.Background(), "server").Result()
	if err != nil {
		c.Detail = "连接失败: " + err.Error()
		return c
	}
	version := infoField(info, "redis_version")
	if !versionAtLeast(version, minRedis) {
		c.Detail = version + " (需要 " + minRedis + " 以上)"
		return c
	}
	c.OK, c.Detail = true, version
	return c
}

// infoField 从 INFO 输出中取 key:value 的值
func infoField(info, key string) string {
	for _, line := range strings.Split(info, "\n") {
		if v, ok := strings.CutPrefix(strings.TrimSpace(line), key+":"); ok {
			return v
		}
	}
	return ""
}

var versionPattern = regexp.MustCompile(`^(\d+)\.(\d+)(?:\.(\d+))?`)

// versionAtLeast 比较版本字符串开头的 x.y.z (如 8.4.2、5.7.44-log、10.11.6-MariaDB); 无法解析时为 false
func versionAtLeast(version, min string) bool {
	a, ok := parseVersion(version)
	b, _ := parseVersion(min)
	if !ok {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return a[i] > b[i]
		}
	}
	return true
}

func parseVersion(s string) (v [3]int, ok bool) {
	m := versionPattern.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return v, false
	}
	for i := 0; i < 3; i++ {
		if m[i+1] != "" {
			v[i], _ = strconv.Atoi(m[i+1])
		}
	}
	return v, true
}
