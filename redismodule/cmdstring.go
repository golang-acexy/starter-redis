package redismodule

import (
	"context"
	"errors"
	"fmt"
	"github.com/acexy/golang-toolkit/util/json"
	"github.com/redis/go-redis/v9"
)

type typeString struct {
}

var (
	stringType = new(typeString)
)

func StringType() *typeString {
	return stringType
}

func keyString(keyFormat string, keyAppend ...interface{}) string {
	if len(keyAppend) > 0 {
		return fmt.Sprintf(keyFormat, keyAppend...)
	}
	return keyFormat
}

func set(ctx context.Context, key RedisKey, value interface{}, keyAppend ...interface{}) error {
	if value == nil {
		return errors.New("nil value")
	}
	status := redisClient.Set(ctx, keyString(key.KeyFormat, keyAppend...), value, key.Expire)
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

func get(ctx context.Context, key RedisKey, keyAppend ...interface{}) (*redis.StringCmd, error) {
	cmd := redisClient.Get(ctx, keyString(key.KeyFormat, keyAppend...))
	if cmd.Err() != nil {
		if errors.Is(cmd.Err(), redis.Nil) {
			return cmd, nil // wrap nil error
		}
		return cmd, cmd.Err()
	}
	return cmd, nil
}

func mget(ctx context.Context, keys ...string) (*redis.SliceCmd, error) {
	slice := redisClient.MGet(ctx, keys...)
	err := slice.Err()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return slice, nil // wrap nil error
		}
		return nil, err
	}
	return slice, nil
}

// Set 设置字符串
func (*typeString) Set(ctx context.Context, key RedisKey, value string, keyAppend ...interface{}) error {
	return set(ctx, key, []byte(value), keyAppend...)
}

// SetBytes 设置字节数据
func (*typeString) SetBytes(ctx context.Context, key RedisKey, value []byte, keyAppend ...interface{}) error {
	return set(ctx, key, value, keyAppend...)
}

// SetAny 原始RedisClient Set指令
// 适用于设置基本类型 或 该值类型需要实现BinaryMarshaler的复杂结构体
func (*typeString) SetAny(ctx context.Context, key RedisKey, value interface{}, keyAppend ...interface{}) error {
	return set(ctx, key, value, keyAppend...)
}

// SetAnyWithJson 设置其他类型值
// 设置任何类型，将被以json格式进行编码存储
func (*typeString) SetAnyWithJson(ctx context.Context, key RedisKey, value any, keyAppend ...interface{}) error {
	bytes, err := json.ToJsonBytesError(value)
	if err != nil {
		return err
	}
	return set(ctx, key, bytes, keyAppend...)
}

// MSet 批量设置字符串
func (*typeString) MSet(ctx context.Context, data map[string]string) error {
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
	return mset(ctx, array)
}

// MSetWithHashTag 批量设置字符串 用于在集群模式指定hashTag将key分配在同一个hash槽中
func (*typeString) MSetWithHashTag(ctx context.Context, hashTag string, data map[string]string) error {
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
	return mset(ctx, array)
}

// MSetBytes 批量设置字节数据
func (*typeString) MSetBytes(ctx context.Context, data map[string][]byte) error {
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
	return mset(ctx, array)
}

// Get 将指定的key以String类型获取
func (*typeString) Get(ctx context.Context, key RedisKey, keyAppend ...interface{}) (string, error) {
	cmd, err := get(ctx, key, keyAppend...)
	if err != nil {
		return "", err
	}
	return cmd.Val(), err
}

func parseMgetStringValue(cmd *redis.SliceCmd, err error) ([]string, error) {
	if err != nil {
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

// MGet 一次性获取多个String类型的值
func (*typeString) MGet(ctx context.Context, keys ...string) ([]string, error) {
	if len(keys) == 0 {
		return nil, errors.New("nil keys")
	}
	cmd, err := mget(ctx, keys...)
	return parseMgetStringValue(cmd, err)
}

// MGetWithHashTag 一次性获取多个String类型的值
func (*typeString) MGetWithHashTag(ctx context.Context, hashTag string, keys ...string) ([]string, error) {
	if len(keys) == 0 {
		return nil, errors.New("nil keys")
	}
	for index, key := range keys {
		keys[index] = "{" + hashTag + "}" + key
	}
	cmd, err := mget(ctx, keys...)
	return parseMgetStringValue(cmd, err)

}

// MGetBytes 一次性获取多个字节数组的值
func (*typeString) MGetBytes(ctx context.Context, keys ...string) ([][]byte, error) {
	if len(keys) == 0 {
		return nil, errors.New("nil keys")
	}
	slice, err := mget(ctx, keys...)
	v, err := slice.Result()
	if err != nil {
		return nil, err
	}

	k := make([][]byte, len(v))
	for i, d := range v {
		if b, ok := d.([]byte); ok {
			k[i] = b
		} else {
			return nil, errors.New("not a string value")
		}
	}
	return k, nil
}

// GetBytes 以字节形式获取指定的值
func (*typeString) GetBytes(ctx context.Context, key RedisKey, keyAppend ...interface{}) ([]byte, error) {
	cmd, err := get(ctx, key, keyAppend...)
	if err != nil {
		return nil, err
	}
	return cmd.Bytes()
}

// GetAny 以指定类型获取指定值
func (*typeString) GetAny(ctx context.Context, key RedisKey, value any, keyAppend ...interface{}) error {
	cmd, err := get(ctx, key, keyAppend...)
	if err != nil {
		return err
	}
	return cmd.Scan(value)
}

// GetAnyWithJson 以Json反序列化形式获取指定值
func (t *typeString) GetAnyWithJson(ctx context.Context, key RedisKey, value any, keyAppend ...interface{}) error {
	bytes, err := t.GetBytes(ctx, key, keyAppend...)
	if err != nil {
		return err
	}
	return json.ParseJsonError(string(bytes), value)
}
