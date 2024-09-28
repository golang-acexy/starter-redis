package redisstarter

import (
	"context"
	"time"
)

type cmdKey struct {
}

var keyCmd = new(cmdKey)

func KeyCmd() *cmdKey {
	return keyCmd
}

// Del 删除指定的key
func (*cmdKey) Del(key RedisKey, keyAppend ...interface{}) int64 {
	return redisClient.Del(context.Background(), OriginKeyString(key.KeyFormat, keyAppend...)).Val()
}

// MDel 一次性删除多个key
func (*cmdKey) MDel(keys ...string) int64 {
	return redisClient.Del(context.Background(), keys...).Val()
}

// Exists 判断指定的key是否存在
func (*cmdKey) Exists(key RedisKey, keyAppend ...interface{}) bool {
	return redisClient.Exists(context.Background(), OriginKeyString(key.KeyFormat, keyAppend...)).Val() > 0
}

// Expire 设置Key过期时间
func (*cmdKey) Expire(key RedisKey, time time.Duration, keyAppend ...interface{}) bool {
	return redisClient.Expire(context.Background(), OriginKeyString(key.KeyFormat, keyAppend...), time).Val()
}

// Ttl 获取命令过期时间
func (*cmdKey) Ttl(key RedisKey, keyAppend ...interface{}) float64 {
	cmd := redisClient.TTL(context.Background(), OriginKeyString(key.KeyFormat, keyAppend...))
	if cmd.Err() != nil {
		return -3
	}
	return cmd.Val().Seconds()
}
