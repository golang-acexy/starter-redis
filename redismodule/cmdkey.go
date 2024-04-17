package redismodule

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
func (*cmdKey) Del(ctx context.Context, key RedisKey, keyAppend ...interface{}) int64 {
	return redisClient.Del(ctx, OriginKeyString(key.KeyFormat, keyAppend...)).Val()
}

// MDel 一次性删除多个key
func (*cmdKey) MDel(ctx context.Context, keys ...string) int64 {
	return redisClient.Del(ctx, keys...).Val()
}

// Exists 判断指定的key是否存在
func (*cmdKey) Exists(ctx context.Context, key RedisKey, keyAppend ...interface{}) bool {
	return redisClient.Exists(ctx, OriginKeyString(key.KeyFormat, keyAppend...)).Val() > 0
}

// Expire 设置Key过期时间
func (*cmdKey) Expire(ctx context.Context, key string, time time.Duration) bool {
	return redisClient.Expire(ctx, key, time).Val()
}
