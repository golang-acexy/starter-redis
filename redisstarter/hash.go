package redisstarter

import (
	"context"
	"errors"
	"github.com/redis/go-redis/v9"
)

type cmdHash struct {
}

var hashCmd = new(cmdHash)

func HashCmd() *cmdHash {
	return hashCmd
}

func hSet(key RedisKey, value []interface{}, keyAppend ...interface{}) error {
	if value == nil {
		return errors.New("nil value")
	}
	originKey := OriginKeyString(key.KeyFormat, keyAppend...)
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

func hMSet(key RedisKey, value []interface{}, keyAppend ...interface{}) error {
	if len(value) == 0 {
		return errors.New("nil value")
	}
	originKey := OriginKeyString(key.KeyFormat, keyAppend...)
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

func hGet(key RedisKey, name string, keyAppend ...interface{}) *redis.StringCmd {
	return redisClient.HGet(context.Background(), OriginKeyString(key.KeyFormat, keyAppend...), name)
}

func hMGet(key RedisKey, names []string, keyAppend ...interface{}) *redis.SliceCmd {
	return redisClient.HMGet(context.Background(), OriginKeyString(key.KeyFormat, keyAppend...), names...)
}

func hGetAll(key RedisKey, keyAppend ...interface{}) *redis.MapStringStringCmd {
	return redisClient.HGetAll(context.Background(), OriginKeyString(key.KeyFormat, keyAppend...))
}

// HExists 判断Hash类型是否存在key
func (*cmdHash) HExists(key RedisKey, name string, keyAppend ...interface{}) bool {
	return redisClient.HExists(context.Background(), OriginKeyString(key.KeyFormat, keyAppend...), name).Val()
}

// HSet 设置Hash类型的值
func (*cmdHash) HSet(key RedisKey, name, value string, keyAppend ...interface{}) error {
	return hSet(key, []interface{}{name, value}, keyAppend...)
}

// HMSet 一次性设置多个Hash类型的值
func (*cmdHash) HMSet(key RedisKey, data map[string]string, keyAppend ...interface{}) error {
	array := make([]interface{}, len(data)*2)
	index := 0
	for k, v := range data {
		array[index] = k
		index++
		array[index] = v
		index++
	}
	return hMSet(key, array, keyAppend...)
}

// HGet 获取Hash指定key值
func (*cmdHash) HGet(key RedisKey, name string, keyAppend ...interface{}) (string, error) {
	cmd := hGet(key, name, keyAppend...)
	if err := cmd.Err(); err != nil {
		if errors.Is(err, redis.Nil) {
			return "", nil // wrap nil error
		}
		return "", err
	}
	return cmd.Val(), nil
}

// HMGet 一次性获取多个hash指定key值
func (*cmdHash) HMGet(key RedisKey, names []string, keyAppend ...interface{}) ([]string, error) {
	cmd := hMGet(key, names, keyAppend...)
	if err := cmd.Err(); err != nil {
		return nil, err
	}
	result, err := cmd.Result()
	if err != nil {
		if errors.Is(cmd.Err(), redis.Nil) {
			return nil, nil // wrap nil error
		}
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
func (*cmdHash) HGetAll(key RedisKey, keyAppend ...interface{}) (map[string]string, error) {
	cmd := hGetAll(key, keyAppend...)
	if err := cmd.Err(); err != nil {
		if errors.Is(cmd.Err(), redis.Nil) {
			return nil, nil
		}
		return nil, err
	}
	return cmd.Result()
}
