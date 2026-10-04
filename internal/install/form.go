package install

import (
	"net"
	"regexp"
	"strings"
	"time"

	"gomaccms/internal/util"

	"github.com/go-sql-driver/mysql"
)

// Form 安装表单
type Form struct {
	MysqlHost     string `form:"mysql_host"`
	MysqlPort     string `form:"mysql_port"`
	MysqlUser     string `form:"mysql_user"`
	MysqlPassword string `form:"mysql_password"`
	MysqlDB       string `form:"mysql_db"`
	RedisAddr     string `form:"redis_addr"`
	RedisPassword string `form:"redis_password"`
	RedisDB       int    `form:"redis_db"`
	Account       string `form:"account"`
	Email         string `form:"email"`
	Password      string `form:"password"`
	Password2     string `form:"password2"`
	Code          string `form:"code"`
}

// defaultForm 安装页的初始值
func defaultForm() Form {
	return Form{MysqlHost: "127.0.0.1", MysqlPort: "3306", MysqlUser: "root", MysqlDB: "FilmSite", RedisAddr: "127.0.0.1:6379"}
}

var (
	dbNamePattern = regexp.MustCompile(`^[A-Za-z0-9_]{1,64}$`)
	portPattern   = regexp.MustCompile(`^[0-9]{1,5}$`)
)

// DSN 由表单拼出 MYSQL_DSN (密码中的特殊字符由 FormatDSN 处理)
func (f Form) DSN() string {
	cfg := mysql.NewConfig()
	cfg.User, cfg.Passwd = f.MysqlUser, f.MysqlPassword
	cfg.Net, cfg.Addr, cfg.DBName = "tcp", net.JoinHostPort(f.MysqlHost, f.MysqlPort), f.MysqlDB
	cfg.ParseTime, cfg.Loc = true, time.Local
	cfg.Params = map[string]string{"charset": "utf8mb4"}
	return cfg.FormatDSN()
}

// normalize 去掉非密码字段前后的空白
func (f *Form) normalize() {
	for _, p := range []*string{&f.MysqlHost, &f.MysqlPort, &f.MysqlUser, &f.MysqlDB, &f.RedisAddr, &f.Account, &f.Email, &f.Code} {
		*p = strings.TrimSpace(*p)
	}
}

// validate 校验表单, 返回第一条错误 ("" 为通过); MySQL / Redis 由环境变量提供时不校验该组
func (f Form) validate(mysqlFromForm, redisFromForm bool) string {
	if mysqlFromForm {
		switch {
		case f.MysqlHost == "" || f.MysqlUser == "":
			return "请填写 MySQL 主机与用户"
		case !portPattern.MatchString(f.MysqlPort):
			return "MySQL 端口无效"
		case !dbNamePattern.MatchString(f.MysqlDB):
			return "数据库名只能是字母、数字或下划线"
		}
	}
	if redisFromForm && f.RedisAddr == "" {
		return "请填写 Redis 地址"
	}
	switch {
	case !util.ValidAccount(f.Account):
		return "管理员账号只能是 3-30 位字母、数字或下划线"
	case !util.ValidEmail(f.Email):
		return "管理员 Email 格式无效"
	case len(f.Password) < 6:
		return "管理员密码至少 6 位"
	case f.Password != f.Password2:
		return "两次输入的密码不一致"
	}
	return ""
}

// withoutPasswords 重新渲染表单时清空所有密码
func (f Form) withoutPasswords() Form {
	f.MysqlPassword, f.RedisPassword, f.Password, f.Password2 = "", "", "", ""
	return f
}
