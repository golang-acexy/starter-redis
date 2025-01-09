package redisstarter

import (
	"context"
	"errors"
	"fmt"
	"github.com/acexy/golang-toolkit/logger"
	"github.com/acexy/golang-toolkit/util/gob"
	"github.com/redis/go-redis/v9"
)

type cmdString struct {
}

var stringCmd = new(cmdString)

func StringCmd() *cmdString {
	return stringCmd
}

func OriginKeyString(keyFormat string, keyAppend ...interface{}) string {
	if len(keyAppend) > 0 {
		return fmt.Sprintf(keyFormat, keyAppend...)
	}
	return keyFormat
}

func set(key RedisKey, value interface{}, keyAppend ...interface{}) error {
	if value == nil {
		return errors.New("nil value")
	}
	status := redisClient.Set(context.Background(), OriginKeyString(key.KeyFormat, keyAppend...), value, key.Expire)
	err := status.Err()
	if err != nil {
		return err
	}
	return nil
}

func mset(data []interface{}) error {
	status := redisClient.MSet(context.Background(), data)
	err := status.Err()
	if err != nil {
		return err
	}
	return nil
}

func get(key RedisKey, keyAppend ...interface{}) (*redis.StringCmd, error) {
	cmd := redisClient.Get(context.Background(), OriginKeyString(key.KeyFormat, keyAppend...))
	if cmd.Err() != nil {
		if errors.Is(cmd.Err(), redis.Nil) {
			return nil, nil // wrap nil error
		}
		return nil, cmd.Err()
	}
	return cmd, nil
}

func mget(keys ...string) (*redis.SliceCmd, error) {
	slice := redisClient.MGet(context.Background(), keys...)
	err := slice.Err()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil // wrap nil error
		}
		return nil, err
	}
	return slice, nil
}

// Set 设置字符串
func (*cmdString) Set(key RedisKey, value string, keyAppend ...interface{}) error {
	return set(key, []byte(value), keyAppend...)
}

// SetBytes 设置字节数据
func (*cmdString) SetBytes(key RedisKey, value []byte, keyAppend ...interface{}) error {
	return set(key, value, keyAppend...)
}

// SetAny 原始RedisClient Set指令
// 适用于设置基本类型 或 该值类型需要实现BinaryMarshaler的复杂结构体
func (*cmdString) SetAny(key RedisKey, value interface{}, keyAppend ...interface{}) error {
	return set(key, value, keyAppend...)
}

// SetAnyWithJson 设置其他类型值
// 设置任何类型
func (*cmdString) SetAnyWithJson(key RedisKey, value any, keyAppend ...interface{}) error {
	bytes, err := gob.Encode(value)
	if err != nil {
		return err
	}
	return set(key, bytes, keyAppend...)
}

// MSet 批量设置字符串
func (*cmdString) MSet(data map[string]string) error {
	if data == nil || len(data) == 0 {
		return errors.New("nil value")
	}
	array := make([]interface{}, len(data)*2)
	index := 0
	for k, v := range data {
		array[index] = k
		index++
		array[index] = v
		index++
	}
	return mset(array)
}

// MSetWithHashTag 批量设置字符串 用于在集群模式指定hashTag将key分配在同一个hash槽中
func (*cmdString) MSetWithHashTag(hashTag string, data map[string]string) error {
	if data == nil || len(data) == 0 {
		return errors.New("nil value")
	}
	array := make([]interface{}, len(data)*2)
	index := 0
	for k, v := range data {
		array[index] = "{" + hashTag + "}" + k
		index++
		array[index] = v
		index++
	}
	return mset(array)
}

// MSetBytes 批量设置字节数据
func (*cmdString) MSetBytes(data map[string][]byte) error {
	if data == nil || len(data) == 0 {
		return errors.New("nil value")
	}
	array := make([]interface{}, len(data)*2)
	index := 0
	for k, v := range data {
		array[index] = k
		index += 1
		array[index] = v
		index += 1
	}
	return mset(array)
}

// MSetBytesWithHashTag 批量设置字节数据
func (*cmdString) MSetBytesWithHashTag(hashTag string, data map[string][]byte) error {
	if data == nil || len(data) == 0 {
		return errors.New("nil value")
	}
	array := make([]interface{}, len(data)*2)
	index := 0
	for k, v := range data {
		array[index] = "{" + hashTag + "}" + k
		index += 1
		array[index] = v
		index += 1
	}
	return mset(array)
}

// Get 将指定的key以String类型获取
func (*cmdString) Get(key RedisKey, keyAppend ...interface{}) (string, error) {
	cmd, err := get(key, keyAppend...)
	if err != nil || cmd == nil {
		return "", err
	}
	return cmd.Val(), err
}

func parseMGetStringValue(cmd *redis.SliceCmd, err error) ([]string, error) {
	if err != nil || cmd == nil {
		return nil, err
	}
	v, err := cmd.Result()
	if err != nil {
		return nil, err
	}
	k := make([]string, len(v))
	for i, d := range v {
		if d != nil {
			if str, ok := d.(string); ok {
				k[i] = str
			} else {
				return nil, errors.New("not a string value")
			}
		}
	}
	return k, nil
}

func parseMGetBytesValue(cmd *redis.SliceCmd, err error) ([][]byte, error) {
	if err != nil {
		return nil, err
	}
	v, err := cmd.Result()
	if err != nil {
		return nil, err
	}
	k := make([][]byte, len(v))
	for i, d := range v {
		if d != nil {
			k[i] = []byte(d.(string))
		}
	}
	return k, nil
}

// MGet 一次性获取多个String类型的值
func (*cmdString) MGet(keys ...string) ([]string, error) {
	if len(keys) == 0 {
		return nil, errors.New("nil keys")
	}
	return parseMGetStringValue(mget(keys...))
}

// MGetWithHashTag 一次性获取多个String类型的值
func (*cmdString) MGetWithHashTag(hashTag string, keys ...string) ([]string, error) {
	if len(keys) == 0 {
		return nil, errors.New("nil keys")
	}
	for index, key := range keys {
		keys[index] = "{" + hashTag + "}" + key
	}
	return parseMGetStringValue(mget(keys...))
}

// MGetBytes 一次性获取多个字节数组的值
func (*cmdString) MGetBytes(keys ...string) ([][]byte, error) {
	defer func() {
		if err := recover(); err != nil {
			logger.Logrus().Errorf("painc %+v", err)
		}
	}()
	if len(keys) == 0 {
		return nil, errors.New("nil keys")
	}
	return parseMGetBytesValue(mget(keys...))
}

// MGetBytesWithHashTag 一次性获取多个字节数组的值
func (*cmdString) MGetBytesWithHashTag(hashTag string, keys ...string) ([][]byte, error) {
	defer func() {
		if err := recover(); err != nil {
			logger.Logrus().Errorf("painc %+v", err)
		}
	}()
	if len(keys) == 0 {
		return nil, errors.New("nil keys")
	}
	for index, key := range keys {
		keys[index] = "{" + hashTag + "}" + key
	}
	return parseMGetBytesValue(mget(keys...))
}

// GetBytes 以字节形式获取指定的值
func (*cmdString) GetBytes(key RedisKey, keyAppend ...interface{}) ([]byte, error) {
	cmd, err := get(key, keyAppend...)
	if err != nil || cmd == nil {
		return nil, err
	}
	return cmd.Bytes()
}

// GetAny 以指定类型获取指定值
func (*cmdString) GetAny(key RedisKey, value any, keyAppend ...interface{}) error {
	cmd, err := get(key, keyAppend...)
	if err != nil || cmd == nil {
		return err
	}
	return cmd.Scan(value)
}

// GetAnyWithJson 以Json反序列化形式获取指定值
func (t *cmdString) GetAnyWithJson(key RedisKey, value any, keyAppend ...interface{}) error {
	bytes, err := t.GetBytes(key, keyAppend...)
	if err != nil {
		return err
	}
	return gob.Decode(bytes, value)
}
