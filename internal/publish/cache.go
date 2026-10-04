package publish

import (
	"gomaccms/internal/db"
	"time"
)

// Cache 发布中心用到的缓存操作 (正式环境为 Redis; 缓存只是优化, 出错时调用方回退到直接查询)
type Cache interface {
	Get(key string) (string, error)
	Set(key, val string, ttl time.Duration) error
	// DelPrefix 删除 key 以及 key:* (如各分类方案 / 域名的缓存)
	DelPrefix(key string) error
}

type redisCache struct{}

func (redisCache) Get(key string) (string, error) {
	return db.Rdb.Get(db.Cxt, key).Result()
}

func (redisCache) Set(key, val string, ttl time.Duration) error {
	return db.Rdb.Set(db.Cxt, key, val, ttl).Err()
}

func (redisCache) DelPrefix(key string) error {
	keys, err := db.Rdb.Keys(db.Cxt, key+":*").Result()
	if err != nil {
		return err
	}
	return db.Rdb.Del(db.Cxt, append(keys, key)...).Err()
}
