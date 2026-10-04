package main

import (
	"fmt"
	"time"

	"gomaccms/internal/bootstrap"
	"gomaccms/internal/config"
	"gomaccms/internal/db"
	"gomaccms/internal/http/router"
	"gomaccms/internal/view/inertia"
)

func main() {
	// 装配各 feature 的 repository / service 单例
	bootstrap.Wire()
	// 执行初始化前等待 (默认20s, STARTUP_DELAY 可覆盖), 让mysql服务完成初始化指令
	time.Sleep(config.StartupDelay)
	//初始化redis客户端
	if err := db.InitRedisConn(); err != nil {
		panic(err)
	}
	// 初始化mysql
	if err := db.InitMysql(); err != nil {
		panic(err)
	}
	// 初始化 Inertia (admin 后台渲染)
	if err := inertia.Setup(); err != nil {
		panic(err)
	}
	// 启动前先执行数据库内容的初始化工作
	bootstrap.DefaultDataInit()
	// 开启路由监听
	r := router.SetupRouter()
	_ = r.Run(fmt.Sprintf(":%s", config.ListenerPort))
}
