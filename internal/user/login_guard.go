package user

import (
	"fmt"
	"gomaccms/internal/db"
	"time"
)

// 登录失败锁定: 同一账号 + IP 连续失败 loginMaxFails 次, 或同一 IP 失败 loginMaxIpFails 次 (不论账号),
// 在 loginLockWindow 内拒绝该 IP 再尝试登录; 只按账号锁定会让他人故意输错来锁住管理员, 所以带上 IP
const (
	loginMaxFails   = 5
	loginMaxIpFails = 20
	loginLockWindow = 15 * time.Minute
)

var errLoginLocked = fmt.Errorf("登录失败次数过多, 请 %d 分钟后再试", int(loginLockWindow.Minutes()))

func loginFailKeys(account, ip string) (userKey, ipKey string) {
	return fmt.Sprintf("Login:Fail:%s:%s", ip, account), fmt.Sprintf("Login:Fail:%s", ip)
}

// loginLocked 是否因失败次数过多而暂时禁止登录
func loginLocked(account, ip string) bool {
	userKey, ipKey := loginFailKeys(account, ip)
	n, _ := db.Rdb.Get(db.Cxt, userKey).Int()
	m, _ := db.Rdb.Get(db.Cxt, ipKey).Int()
	return n >= loginMaxFails || m >= loginMaxIpFails
}

// recordLoginFail 记录一次登录失败, 返回给用户的错误 (达到上限时提示已锁定, 否则提示剩余次数)
func recordLoginFail(account, ip string, err error) error {
	userKey, ipKey := loginFailKeys(account, ip)
	n := incrWithin(userKey)
	m := incrWithin(ipKey)
	if n >= loginMaxFails || m >= loginMaxIpFails {
		return errLoginLocked
	}
	return fmt.Errorf("%s, 再失败 %d 次将锁定 %d 分钟", err.Error(), min(loginMaxFails-n, loginMaxIpFails-m), int(loginLockWindow.Minutes()))
}

// clearLoginFail 登录成功后清除该账号 + IP 的失败次数
func clearLoginFail(account, ip string) {
	userKey, _ := loginFailKeys(account, ip)
	db.Rdb.Del(db.Cxt, userKey)
}

// incrWithin 计数加一; 第一次失败时开始计算锁定时间窗口
func incrWithin(key string) int {
	n, err := db.Rdb.Incr(db.Cxt, key).Result()
	if err != nil {
		return 0
	}
	if n == 1 {
		db.Rdb.Expire(db.Cxt, key, loginLockWindow)
	}
	return int(n)
}
