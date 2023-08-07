package redismodule

import (
	"context"
	"errors"
	"github.com/acexy/golang-toolkit/util"
	"github.com/redis/go-redis/v9"
	"time"
)

// RawClient 获取原始RedisClient进行操作
func RawClient() redis.UniversalClient {
	return redisClient
}

func set(ctx context.Context, key RedisKey, value interface{}, expiration ...time.Duration) error {
	if value == nil {
		return errors.New("nil value")
	}
	var expire time.Duration
	if len(expiration) == 0 {
		expire = 0
	} else {
		expire = expiration[0]
	}
	status := redisClient.Set(ctx, string(key), value, expire)
	err := status.Err()
	if err != nil {
		return err
	}
	return nil
}

// Set 设置字符串
func Set(ctx context.Context, key RedisKey, value string, expiration ...time.Duration) error {
	return set(ctx, key, []byte(value), expiration...)
}

// SetBytes 设置字节
func SetBytes(ctx context.Context, key RedisKey, value []byte, expiration ...time.Duration) error {
	return set(ctx, key, value, expiration...)
}

// SetAny 原始RedisClient Set指令
// 适用于设置基本类型&实现BinaryMarshaler的复杂结构体
func SetAny(ctx context.Context, key RedisKey, value interface{}, expiration ...time.Duration) error {
	return set(ctx, key, value, expiration...)
}

// SetAnyWithJson 设置其他类型值
// 设置任何类型，将被以json格式进行编码存储
func SetAnyWithJson(ctx context.Context, key RedisKey, value any, expiration ...time.Duration) error {
	bytes, err := util.ToJsonBytesError(value)
	if err != nil {
		return err
	}
	return set(ctx, key, bytes, expiration...)
}

func get(ctx context.Context, key RedisKey) (*redis.StringCmd, error) {
	cmd := redisClient.Get(ctx, string(key))
	if cmd.Err() != nil {
		return cmd, cmd.Err()
	}
	return cmd, nil
}

func GetString(ctx context.Context, key RedisKey) (string, error) {
	cmd, err := get(ctx, key)
	if err != nil {
		return "", err
	}
	return cmd.Result()
}

func GetBytes(ctx context.Context, key RedisKey) ([]byte, error) {
	cmd, err := get(ctx, key)
	if err != nil {
		return nil, err
	}
	return cmd.Bytes()
}

func GetAny(ctx context.Context, key RedisKey, object any) error {
	cmd, err := get(ctx, key)
	if err != nil {
		return err
	}
	return cmd.Scan(object)
}

func GetAnyWithJson(ctx context.Context, key RedisKey, object any) error {
	bytes, err := GetBytes(ctx, key)
	if err != nil {
		return err
	}
	return util.ParseJsonError(string(bytes), object)
}
