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
func (*cmdKey) Del(key RedisKey, keyAppend ...any) int64 {
	return rawRedisClient().Del(context.Background(), key.RawKeyString(keyAppend...)).Val()
}

// MDel 一次性删除多个key
func (*cmdKey) MDel(keys ...string) int64 {
	return rawRedisClient().Del(context.Background(), keys...).Val()
}

// Exists 判断指定的key是否存在
func (*cmdKey) Exists(key RedisKey, keyAppend ...any) bool {
	return rawRedisClient().Exists(context.Background(), key.RawKeyString(keyAppend...)).Val() > 0
}

// Expire 设置Key过期时间
func (*cmdKey) Expire(key RedisKey, time time.Duration, keyAppend ...any) bool {
	return rawRedisClient().Expire(context.Background(), key.RawKeyString(keyAppend...), time).Val()
}

// Ttl 获取命令过期时间
func (*cmdKey) Ttl(key RedisKey, keyAppend ...any) float64 {
	cmd := rawRedisClient().TTL(context.Background(), key.RawKeyString(keyAppend...))
	if cmd.Err() != nil {
		return -3
	}
	return cmd.Val().Seconds()
}
