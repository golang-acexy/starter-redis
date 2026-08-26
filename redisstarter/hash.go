package redisstarter

import (
	"context"

	"github.com/acexy/golang-toolkit/util/gob"
	"github.com/redis/go-redis/v9"
)

type cmdHash struct {
}

var hashCmd = new(cmdHash)

func HashCmd() *cmdHash {
	return hashCmd
}

func hSet(key RedisKey, value []any, keyAppend ...any) error {
	if value == nil {
		return ErrNilValue
	}
	originKey := key.RawKeyString(keyAppend...)
	cmd := redisClient.HSet(context.Background(), originKey, value)
	if cmd.Err() != nil {
		return cmd.Err()
	}
	if key.Expire > 0 {
		if keyCmd.Ttl(key, keyAppend...) < 0 {
			keyCmd.Expire(key, key.Expire, keyAppend...)
		}
	}
	return nil
}

func hMSet(key RedisKey, value []any, keyAppend ...any) error {
	if len(value) == 0 {
		return ErrNilValue
	}
	originKey := key.RawKeyString(keyAppend...)
	cmd := redisClient.HMSet(context.Background(), originKey, value)
	if cmd.Err() != nil {
		return cmd.Err()
	}
	if key.Expire > 0 {
		if keyCmd.Ttl(key, keyAppend...) < 0 {
			keyCmd.Expire(key, key.Expire, keyAppend...)
		}
	}
	return nil
}

func hGet(key RedisKey, name string, keyAppend ...any) *redis.StringCmd {
	return redisClient.HGet(context.Background(), key.RawKeyString(keyAppend...), name)
}

func hMGet(key RedisKey, names []string, keyAppend ...any) *redis.SliceCmd {
	return redisClient.HMGet(context.Background(), key.RawKeyString(keyAppend...), names...)
}

func hGetAll(key RedisKey, keyAppend ...any) *redis.MapStringStringCmd {
	return redisClient.HGetAll(context.Background(), key.RawKeyString(keyAppend...))
}

// HExists 判断Hash类型是否存在key
func (*cmdHash) HExists(key RedisKey, name string, keyAppend ...any) bool {
	return redisClient.HExists(context.Background(), key.RawKeyString(keyAppend...), name).Val()
}

// HSet 设置Hash类型的值
func (*cmdHash) HSet(key RedisKey, name, value string, keyAppend ...any) error {
	return hSet(key, []any{name, value}, keyAppend...)
}

// HSetAny 设置Hash类型的值 任何值类型
func (*cmdHash) HSetAny(key RedisKey, name string, value any, keyAppend ...any) error {
	encode, err := gob.Encode(value)
	if err != nil {
		return err
	}
	return hSet(key, []any{name, encode}, keyAppend...)
}

// HGetAny 获取Hash指定key值 任何值类型
func (*cmdHash) HGetAny(key RedisKey, name string, result any, keyAppend ...any) error {
	cmd := hGet(key, name, keyAppend...)
	if cmd.Err() != nil {
		return cmd.Err()
	}
	bytes, err := cmd.Bytes()
	if err != nil {
		return err
	}
	return gob.Decode(bytes, result)
}

// HMSet 一次性设置多个Hash类型的值
func (*cmdHash) HMSet(key RedisKey, data map[string]string, keyAppend ...any) error {
	array := make([]any, len(data)*2)
	index := 0
	for k, v := range data {
		array[index] = k
		index++
		array[index] = v
		index++
	}
	return hMSet(key, array, keyAppend...)
}

// HMSetAny 一次性设置多个Hash类型的值 任意值类型
func (*cmdHash) HMSetAny(key RedisKey, data map[string]any, keyAppend ...any) error {
	array := make([]any, len(data)*2)
	index := 0
	for k, v := range data {
		array[index] = k
		index++
		b, err := gob.Encode(v)
		if err != nil {
			return err
		}
		array[index] = b
		index++
	}
	return hMSet(key, array, keyAppend...)
}

// HGet 获取Hash指定key值
func (*cmdHash) HGet(key RedisKey, name string, keyAppend ...any) (string, error) {
	cmd := hGet(key, name, keyAppend...)
	if err := cmd.Err(); err != nil {
		return "", err
	}
	return cmd.Val(), nil
}

// HMGet 一次性获取多个hash指定key值
func (*cmdHash) HMGet(key RedisKey, names []string, keyAppend ...any) ([]string, error) {
	cmd := hMGet(key, names, keyAppend...)
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
			str, ok := v.(string)
			if !ok {
				return nil, ErrNotStringValue
			}
			m[i] = str
		}
	}
	return m, nil
}

// HGetAll 获取指定key中所有数据
func (*cmdHash) HGetAll(key RedisKey, keyAppend ...any) (map[string]string, error) {
	cmd := hGetAll(key, keyAppend...)
	if err := cmd.Err(); err != nil {
		return nil, err
	}
	return cmd.Result()
}
