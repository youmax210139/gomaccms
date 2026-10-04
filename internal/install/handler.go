package install

import (
	"context"
	"crypto/rand"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"log"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"gomaccms/database/migrations"
	"gomaccms/database/seed"
	"gomaccms/internal/config"
	"gomaccms/internal/db"
	"gomaccms/internal/http/response"

	"github.com/gin-gonic/gin"
	"github.com/go-sql-driver/mysql"
)

//go:embed templates/install.html
var templateFS embed.FS

var pageTmpl = template.Must(template.ParseFS(templateFS, "templates/install.html"))

// Handler 安装模式下的全部路由
type Handler struct {
	sw     *Switch
	boot   func() (http.Handler, error)
	code   string
	mu     sync.Mutex // 同时只能有一个安装在执行
	engine *gin.Engine
}

// NewHandler 建立安装路由并在日志打印安装码; 安装成功后用 boot 的结果替换 sw 中的 handler
func NewHandler(sw *Switch, boot func() (http.Handler, error)) *Handler {
	h := &Handler{sw: sw, boot: boot, code: newCode()}
	log.Printf("install: 尚未安装, 请打开网站 /install 完成安装, 安装码: %s", h.code)
	e := gin.New()
	e.Use(gin.Logger(), gin.Recovery())
	e.GET("/install", func(c *gin.Context) { h.render(c, http.StatusOK, defaultForm(), "") })
	e.POST("/install", h.submit)
	e.POST("/install/test", h.test)
	e.NoRoute(func(c *gin.Context) { c.Redirect(http.StatusFound, "/install") })
	h.engine = e
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) { h.engine.ServeHTTP(w, r) }

// newCode 6 位随机数字
func newCode() string {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%06d", n.Int64())
}

type pageData struct {
	Checks       []Check
	Blocked      string
	MysqlFromEnv bool
	RedisFromEnv bool
	Form         Form
	Error        string
}

// render 安装页: 环境检查 (+ 环境变量提供的 MySQL / Redis 检查) 与表单
func (h *Handler) render(c *gin.Context, status int, f Form, errMsg string) {
	d := pageData{
		Checks:       CheckEnvironment(),
		MysqlFromEnv: config.EnvProvided("MYSQL_DSN"),
		RedisFromEnv: config.EnvProvided("REDIS_ADDR"),
		Form:         f.withoutPasswords(),
		Error:        errMsg,
	}
	if d.MysqlFromEnv {
		d.Checks = append(d.Checks, CheckMysql(config.MysqlDsn))
	}
	if d.RedisFromEnv {
		d.Checks = append(d.Checks, CheckRedis(config.RedisAddr, config.RedisPassword, config.RedisDBNo))
	}
	d.Blocked = Blocked(d.Checks)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(status)
	if err := pageTmpl.Execute(c.Writer, d); err != nil {
		log.Printf("install: 渲染安装页: %v", err)
	}
}

func (h *Handler) bindForm(c *gin.Context) Form {
	var f Form
	_ = c.ShouldBind(&f) // 字段缺失 / 格式错误交给 validate
	f.normalize()
	return f
}

func (h *Handler) submit(c *gin.Context) {
	f := h.bindForm(c)
	if !h.mu.TryLock() {
		h.render(c, http.StatusConflict, f, "安装正在进行中, 请稍候")
		return
	}
	defer h.mu.Unlock()
	if err := h.install(f); err != nil {
		log.Printf("install: 安装失败: %v", err)
		h.render(c, http.StatusOK, f, err.Error())
		return
	}
	log.Printf("install: 安装完成, 管理员 %s", f.Account)
	c.Redirect(http.StatusFound, "/login")
}

// test 「测试连接」: 检查表单里的 MySQL / Redis (环境变量提供的组不在此检查)
func (h *Handler) test(c *gin.Context) {
	f := h.bindForm(c)
	if f.Code != h.code {
		response.Failed("安装码错误 (见服务启动日志)", c)
		return
	}
	data := gin.H{}
	if !config.EnvProvided("MYSQL_DSN") {
		data["mysql"] = CheckMysql(f.DSN())
	}
	if !config.EnvProvided("REDIS_ADDR") {
		data["redis"] = CheckRedis(f.RedisAddr, f.RedisPassword, f.RedisDB)
	}
	response.Success(data, "", c)
}

// install 执行安装; 任一步失败返回错误, 不写 lock, 可直接重试
func (h *Handler) install(f Form) error {
	if f.Code != h.code {
		return errors.New("安装码错误 (见服务启动日志)")
	}
	mysqlFromForm, redisFromForm := !config.EnvProvided("MYSQL_DSN"), !config.EnvProvided("REDIS_ADDR")
	if msg := f.validate(mysqlFromForm, redisFromForm); msg != "" {
		return errors.New(msg)
	}
	if msg := Blocked(CheckEnvironment()); msg != "" {
		return errors.New(msg)
	}

	values := map[string]string{}
	dsn := config.MysqlDsn
	if mysqlFromForm {
		dsn = f.DSN()
		values["MYSQL_DSN"] = dsn
	}
	addr, password, dbNo := config.RedisAddr, config.RedisPassword, config.RedisDBNo
	if redisFromForm {
		addr, password, dbNo = f.RedisAddr, f.RedisPassword, f.RedisDB
		values["REDIS_ADDR"], values["REDIS_PASSWORD"], values["REDIS_DB"] = addr, password, strconv.Itoa(dbNo)
	}
	if msg := Blocked([]Check{CheckMysql(dsn), CheckRedis(addr, password, dbNo)}); msg != "" {
		return errors.New(msg)
	}
	if err := ensureDatabase(dsn); err != nil {
		return err
	}

	if err := config.WriteEnvFile(config.EnvFilePath, values); err != nil {
		return fmt.Errorf("写入 %s: %w", config.EnvFilePath, err)
	}
	config.Load()
	if err := db.InitRedisConn(); err != nil {
		return fmt.Errorf("连接 Redis: %w", err)
	}
	if err := db.InitMysql(); err != nil {
		return fmt.Errorf("连接 MySQL: %w", err)
	}
	if err := migrations.Up(context.Background()); err != nil {
		return err
	}
	if err := seed.CreateAdmin(f.Account, f.Email, f.Password); err != nil {
		return fmt.Errorf("建立管理员: %w", err)
	}
	handler, err := safeBoot(h.boot)
	if err != nil {
		return err
	}
	if err := WriteLock(); err != nil {
		return fmt.Errorf("写入 %s: %w", LockPath, err)
	}
	h.sw.Set(handler)
	return nil
}

// safeBoot 执行 boot, 把其中的 panic (DefaultDataInit 使用 log.Panic) 转成错误
func safeBoot(boot func() (http.Handler, error)) (h http.Handler, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("初始化失败: %v", r)
		}
	}()
	return boot()
}

// ensureDatabase 库不存在 (MySQL 1049) 时建立; 其他连接错误已由 CheckMysql 报告, 此处忽略
func ensureDatabase(dsn string) error {
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return err
	}
	cfg.Timeout = 5 * time.Second
	name := cfg.DBName
	pingErr := ping(cfg.FormatDSN())
	var me *mysql.MySQLError
	if pingErr == nil || !errors.As(pingErr, &me) || me.Number != 1049 {
		return nil
	}
	cfg.DBName = ""
	conn, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return err
	}
	defer conn.Close()
	stmt := "CREATE DATABASE IF NOT EXISTS `" + strings.ReplaceAll(name, "`", "") + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci"
	if _, err := conn.Exec(stmt); err != nil {
		return fmt.Errorf("数据库 %s 不存在且无法自动创建 (%v), 请先手动创建", name, err)
	}
	log.Printf("install: 已建立数据库 %s", name)
	return nil
}

func ping(dsn string) error {
	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	defer conn.Close()
	return conn.Ping()
}
