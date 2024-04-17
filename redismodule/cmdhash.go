package redismodule

import (
	"context"
	"errors"
	"github.com/acexy/golang-toolkit/util/json"
	"github.com/redis/go-redis/v9"
)

type cmdHash struct {
}

var hashCmd = new(cmdHash)

func HashCmd() *cmdHash {
	return hashCmd
}

func hSet(ctx context.Context, key RedisKey, value []interface{}, keyAppend ...interface{}) error {
	if value == nil {
		return errors.New("nil value")
	}
	originKey := OriginKeyString(key.KeyFormat, keyAppend...)
	cmd := redisClient.HSet(ctx, originKey, value)
	if cmd.Err() != nil {
		return cmd.Err()
	}
	if key.Expire > 0 {
		keyCmd.Expire(ctx, originKey, key.Expire)
	}
	return nil
}

func hMSet(ctx context.Context, key RedisKey, value []interface{}, keyAppend ...interface{}) error {
	if len(value) == 0 {
		return errors.New("nil value")
	}
	originKey := OriginKeyString(key.KeyFormat, keyAppend...)
	cmd := redisClient.HMSet(ctx, originKey, value)
	if cmd.Err() != nil {
		return cmd.Err()
	}
	if key.Expire > 0 {
		keyCmd.Expire(ctx, originKey, key.Expire)
	}
	return nil
}

func hGet(ctx context.Context, key RedisKey, name string, keyAppend ...interface{}) *redis.StringCmd {
	return redisClient.HGet(ctx, OriginKeyString(key.KeyFormat, keyAppend...), name)
}

func hMGet(ctx context.Context, key RedisKey, names []string, keyAppend ...interface{}) *redis.SliceCmd {
	return redisClient.HMGet(ctx, OriginKeyString(key.KeyFormat, keyAppend...), names...)
}

func hGetAll(ctx context.Context, key RedisKey, keyAppend ...interface{}) *redis.MapStringStringCmd {
	return redisClient.HGetAll(ctx, OriginKeyString(key.KeyFormat, keyAppend...))
}

// HSet 设置Hash类型的值
func (*cmdHash) HSet(ctx context.Context, key RedisKey, name, value string, keyAppend ...interface{}) error {
	return hSet(ctx, key, []interface{}{name, value}, keyAppend...)
}

// HMSet 一次性设置多个Hash类型的值
func (*cmdHash) HMSet(ctx context.Context, key RedisKey, data map[string]string, keyAppend ...interface{}) error {
	array := make([]interface{}, len(data)*2)
	index := 0
	for k, v := range data {
		array[index] = k
		index++
		array[index] = v
		index++
	}
	return hMSet(ctx, key, array, keyAppend...)
}

// HGet 获取Hash指定key值
func (*cmdHash) HGet(ctx context.Context, key RedisKey, name string, keyAppend ...interface{}) (string, error) {
	cmd := hGet(ctx, key, name, keyAppend...)
	if err := cmd.Err(); err != nil {
		if errors.Is(err, redis.Nil) {
			return "", nil // wrap nil error
		}
		return "", err
	}
	return cmd.Val(), nil
}

// HMGet 一次性获取多个hash指定key值
func (*cmdHash) HMGet(ctx context.Context, key RedisKey, names []string, keyAppend ...interface{}) ([]string, error) {
	cmd := hMGet(ctx, key, names, keyAppend...)
	if err := cmd.Err(); err != nil {
		return nil, err
	}
	result, err := cmd.Result()
	if err != nil {
		return nil, err
	}
	m := make([]string, len(result))
	for i, v := range result {
		if v != nil {
			m[i] = v.(string)
		}
	}
	return m, nil
}

// HGetAll 获取指定key中所有数据
func (*cmdHash) HGetAll(ctx context.Context, key RedisKey, keyAppend ...interface{}) (map[string]string, error) {
	cmd := hGetAll(ctx, key, keyAppend...)
	if err := cmd.Err(); err != nil {
		return nil, err
	}
	return cmd.Result()
}

// HSetAnyWithJson 设置Hash类型的值 json格式序列化
func (*cmdHash) HSetAnyWithJson(ctx context.Context, key RedisKey, name string, value interface{}, keyAppend ...interface{}) error {
	return hSet(ctx, key, []interface{}{name, json.ToJsonBytes(value)}, keyAppend...)
}

// HGetAnyWithJson 获取Hash类型的值 json格式反序列化
func (*cmdHash) HGetAnyWithJson(ctx context.Context, key RedisKey, name string, value any, keyAppend ...interface{}) error {
	cmd := hGet(ctx, key, name, keyAppend...)
	if cmd.Err() != nil {
		if errors.Is(cmd.Err(), redis.Nil) {
			return nil // wrap nil error
		}
		return cmd.Err()
	}
	bytes, err := cmd.Bytes()
	if err != nil {
		return err
	}
	return json.ParseBytesError(bytes, value)
}
