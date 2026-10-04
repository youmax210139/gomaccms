package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"gomaccms/internal/bootstrap"
	"gomaccms/internal/config"
	"gomaccms/internal/db"
	"gomaccms/internal/http/router"
	"gomaccms/internal/install"
	"gomaccms/internal/view/inertia"
)

func main() {
	// 装配各 feature 的 repository / service 单例 (配置已由 config 包 init 时 Load)
	bootstrap.Wire()
	// 执行初始化前等待 (默认20s, STARTUP_DELAY 可覆盖), 让mysql服务完成初始化指令
	time.Sleep(config.StartupDelay)

	sw := &install.Switch{}
	installed := install.Locked()
	if !installed {
		ok, err := install.Detect()
		if err != nil && install.Configured() {
			// 已有数据库配置却连不上: 照旧报错退出 (由进程管理器重启), 不让已上线的网站进入安装模式
			panic(fmt.Errorf("连接 MySQL 失败: %w", err))
		}
		installed = ok
	}
	if installed {
		if err := db.InitRedisConn(); err != nil {
			panic(err)
		}
		if err := db.InitMysql(); err != nil {
			panic(err)
		}
		h, err := boot()
		if err != nil {
			panic(err)
		}
		sw.Set(h)
	} else {
		// 未安装: 先挂安装页, 安装成功后由安装流程执行 boot 并换成完整路由
		sw.Set(install.NewHandler(sw, boot))
	}

	addr := fmt.Sprintf(":%s", config.ListenerPort)
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, sw); err != nil {
		log.Fatal(err)
	}
}

// boot 在 Redis / MySQL 已连接后初始化 Inertia 与数据库内容 (迁移、seed、采集与定时任务), 返回完整路由
func boot() (http.Handler, error) {
	if err := inertia.Setup(); err != nil {
		return nil, err
	}
	bootstrap.DefaultDataInit()
	return router.SetupRouter().Handler(), nil
}
