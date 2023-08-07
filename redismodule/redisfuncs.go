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

func convertKeyType(keys ...RedisKey) []string {
	k := make([]string, len(keys))
	for i, v := range keys {
		k[i] = string(v)
	}
	return k
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

func mset(ctx context.Context, data []interface{}) error {
	status := redisClient.MSet(ctx, data)
	err := status.Err()
	if err != nil {
		return err
	}
	return nil
}

func get(ctx context.Context, key RedisKey) (*redis.StringCmd, error) {
	cmd := redisClient.Get(ctx, string(key))
	if cmd.Err() != nil {
		return cmd, cmd.Err()
	}
	return cmd, nil
}

func mget(ctx context.Context, keys ...string) (*redis.SliceCmd, error) {
	slice := redisClient.MGet(ctx, keys...)
	err := slice.Err()
	if err != nil {
		return nil, err
	}
	return slice, nil
}

// Set 设置字符串
func Set(ctx context.Context, key RedisKey, value string, expiration ...time.Duration) error {
	return set(ctx, key, []byte(value), expiration...)
}

// MSet 批量设置字符串
func MSet(ctx context.Context, data map[RedisKey]string) error {
	if data == nil || len(data) == 0 {
		return errors.New("nil value")
	}
	if len(data)%2 != 0 {
		return errors.New("bad value length")
	}
	array := make([]interface{}, len(data)*2)
	index := 0
	for k, v := range data {
		array[index] = string(k)
		index += 1
		array[index] = v
		index += 1
	}
	return mset(ctx, array)
}

// SetBytes 设置字节
func SetBytes(ctx context.Context, key RedisKey, value []byte, expiration ...time.Duration) error {
	return set(ctx, key, value, expiration...)
}

func MSetBytes(ctx context.Context, data map[RedisKey][]byte) error {
	if data == nil || len(data) == 0 {
		return errors.New("nil value")
	}
	array := make([]interface{}, len(data)*2)
	index := 0
	for k, v := range data {
		array[index] = string(k)
		index += 1
		array[index] = v
		index += 1
	}
	return mset(ctx, array)
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

func Get(ctx context.Context, key RedisKey) (string, error) {
	cmd, err := get(ctx, key)
	if err != nil {
		return "", err
	}
	return cmd.String(), nil
}

func MGet(ctx context.Context, keys ...RedisKey) ([]string, error) {
	if len(keys) == 0 {
		return nil, errors.New("nil keys")
	}
	slice, err := mget(ctx, convertKeyType(keys...)...)
	v, err := slice.Result()
	if err != nil {
		return nil, err
	}
	k := make([]string, len(v))
	for i, d := range v {
		if str, ok := d.(string); ok {
			k[i] = str
		} else {
			return nil, errors.New("not a string value")
		}
	}
	return k, nil
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

func MGetAny(ctx context.Context, object any, keys ...RedisKey) error {
	if len(keys) == 0 {
		return errors.New("nil keys")
	}
	slice, err := mget(ctx, convertKeyType(keys...)...)
	if err != nil {
		return err
	}
	err = slice.Scan(object)
	if err != nil {
		return err
	}
	return nil
}
