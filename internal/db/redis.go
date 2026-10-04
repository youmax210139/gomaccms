package db

import (
	"context"
	"gomaccms/internal/config"
	"time"

	"github.com/redis/go-redis/v9"
)

/*
redis 工具类
*/
var Rdb *redis.Client
var Cxt = context.Background()

// InitRedisConn 初始化redis客户端
func InitRedisConn() error {

	Rdb = redis.NewClient(&redis.Options{
		Addr:        config.RedisAddr,
		Password:    config.RedisPassword,
		DB:          config.RedisDBNo,
		PoolSize:    10,               // 最大连接数
		DialTimeout: time.Second * 10, // 超时时间
	})
	// 测试连接是否正常 (失败时返回错误: 启动时由 main panic, 安装向导则显示给用户)
	return Rdb.Ping(Cxt).Err()
}
