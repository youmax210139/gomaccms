// Package install 安装向导: 首次启动时在浏览器里完成环境检查、连接配置、迁移与初始数据,
// 完成后写入 storage/install.lock, 之后启动不再进入安装流程。
package install

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"gomaccms/internal/config"

	"github.com/go-sql-driver/mysql"
)

// LockPath 安装完成标记
const LockPath = "./storage/install.lock"

// Locked 是否已安装 (lock 存在)
func Locked() bool {
	_, err := os.Stat(LockPath)
	return err == nil
}

// WriteLock 写入安装完成标记 (内容为安装时间)
func WriteLock() error {
	if err := os.MkdirAll(filepath.Dir(LockPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(LockPath, []byte("installed at "+time.Now().Format(time.RFC3339)+"\n"), 0o644)
}

// Detect 没有 lock 时判断是否为已在运行的旧部署: 数据库能连上、goose 有迁移记录且已有管理员。
// 为真时补写 lock。连接或查询失败一律视为未安装, 不 panic。
func Detect() bool {
	cfg, err := mysql.ParseDSN(config.MysqlDsn)
	if err != nil {
		return false
	}
	cfg.Timeout = 5 * time.Second
	conn, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return false
	}
	defer conn.Close()
	var migrated, users int
	if conn.QueryRow("SELECT COUNT(*) FROM goose_db_version WHERE version_id > 0").Scan(&migrated) != nil || migrated == 0 {
		return false
	}
	if conn.QueryRow("SELECT COUNT(*) FROM users").Scan(&users) != nil || users == 0 {
		return false
	}
	if err := WriteLock(); err != nil {
		log.Printf("install: 补写 %s 失败: %v", LockPath, err)
	} else {
		log.Printf("install: 数据库已安装, 已补写 %s", LockPath)
	}
	return true
}

// Switch 可原子替换的 http.Handler: 未安装时是安装页, 安装完成后换成完整路由
type Switch struct {
	h atomic.Pointer[http.Handler]
}

// Set 替换当前 handler
func (s *Switch) Set(h http.Handler) { s.h.Store(&h) }

func (s *Switch) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	(*s.h.Load()).ServeHTTP(w, r)
}
