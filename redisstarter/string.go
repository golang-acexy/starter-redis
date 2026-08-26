package redisstarter

import (
	"context"

	"github.com/acexy/golang-toolkit/util/coll"
	"github.com/acexy/golang-toolkit/util/gob"
	"github.com/redis/go-redis/v9"
)

type cmdString struct {
}

var stringCmd = new(cmdString)

func StringCmd() *cmdString {
	return stringCmd
}

func set(key RedisKey, value any, keyAppend ...any) error {
	if value == nil {
		return ErrNilValue
	}
	status := redisClient.Set(context.Background(), key.RawKeyString(keyAppend...), value, key.Expire)
	err := status.Err()
	if err != nil {
		return err
	}
	return nil
}

func mset(data []any) error {
	status := redisClient.MSet(context.Background(), data)
	err := status.Err()
	if err != nil {
		return err
	}
	return nil
}

func get(key RedisKey, keyAppend ...any) (*redis.StringCmd, error) {
	cmd := redisClient.Get(context.Background(), key.RawKeyString(keyAppend...))
	if cmd.Err() != nil {
		return nil, cmd.Err()
	}
	return cmd, nil
}

func mget(keys ...string) (*redis.SliceCmd, error) {
	slice := redisClient.MGet(context.Background(), keys...)
	err := slice.Err()
	if err != nil {
		return nil, err
	}
	return slice, nil
}

// Set 设置字符串
func (*cmdString) Set(key RedisKey, value string, keyAppend ...any) error {
	return set(key, []byte(value), keyAppend...)
}

// SetBytes 设置字节数据
func (*cmdString) SetBytes(key RedisKey, value []byte, keyAppend ...any) error {
	return set(key, value, keyAppend...)
}

// SetAny 原始RedisClient Set指令
// 适用于设置基本类型 或 该值类型需要实现BinaryMarshaler的复杂结构体
func (*cmdString) SetAny(key RedisKey, value any, keyAppend ...any) error {
	return set(key, value, keyAppend...)
}

// SetAnyWithGob 设置其他类型值
// 设置任何类型
func (*cmdString) SetAnyWithGob(key RedisKey, value any, keyAppend ...any) error {
	bytes, err := gob.Encode(value)
	if err != nil {
		return err
	}
	return set(key, bytes, keyAppend...)
}

// MSet 批量设置字符串
func (*cmdString) MSet(data map[string]string) error {
	if data == nil || len(data) == 0 {
		return ErrNilValue
	}
	array := make([]any, len(data)*2)
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
		return ErrNilValue
	}
	array := make([]any, len(data)*2)
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
		return ErrNilValue
	}
	array := make([]any, len(data)*2)
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
		return ErrNilValue
	}
	array := make([]any, len(data)*2)
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
func (*cmdString) Get(key RedisKey, keyAppend ...any) (string, error) {
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
				return nil, ErrNotStringValue
			}
		}
	}
	return k, nil
}

func parseMGetBytesValue(cmd *redis.SliceCmd, err error) ([][]byte, error) {
	if err != nil || cmd == nil {
		return nil, err
	}
	v, err := cmd.Result()
	if err != nil {
		return nil, err
	}
	k := make([][]byte, len(v))
	for i, d := range v {
		if d != nil {
			str, ok := d.(string)
			if !ok {
				return nil, ErrNotStringValue
			}
			k[i] = []byte(str)
		}
	}
	return k, nil
}

// MGet 一次性获取多个String类型的值
func (*cmdString) MGet(keys ...string) ([]string, error) {
	if len(keys) == 0 {
		return nil, ErrNilKeys
	}
	return parseMGetStringValue(mget(keys...))
}

// MGetWithHashTag 一次性获取多个String类型的值
func (*cmdString) MGetWithHashTag(hashTag string, keys ...string) ([]string, error) {
	if len(keys) == 0 {
		return nil, ErrNilKeys
	}
	taggedKeys := withHashTag(hashTag, keys)
	return parseMGetStringValue(mget(taggedKeys...))
}

// MGetBytes 一次性获取多个字节数组的值
func (*cmdString) MGetBytes(keys ...string) ([][]byte, error) {
	if len(keys) == 0 {
		return nil, ErrNilKeys
	}
	return parseMGetBytesValue(mget(keys...))
}

// MGetBytesWithHashTag 一次性获取多个字节数组的值
func (*cmdString) MGetBytesWithHashTag(hashTag string, keys ...string) ([][]byte, error) {
	if len(keys) == 0 {
		return nil, ErrNilKeys
	}
	taggedKeys := withHashTag(hashTag, keys)
	return parseMGetBytesValue(mget(taggedKeys...))
}

func withHashTag(hashTag string, keys []string) []string {
	return coll.SliceCollect(keys, func(key string) string {
		return "{" + hashTag + "}" + key
	})
}

// GetBytes 以字节形式获取指定的值
func (*cmdString) GetBytes(key RedisKey, keyAppend ...any) ([]byte, error) {
	cmd, err := get(key, keyAppend...)
	if err != nil || cmd == nil {
		return nil, err
	}
	return cmd.Bytes()
}

// GetAny 以指定类型获取指定值
// 适用于设置基本类型 或 该值类型需要实现BinaryUnmarshaler的复杂结构体
func (*cmdString) GetAny(key RedisKey, value any, keyAppend ...any) error {
	cmd, err := get(key, keyAppend...)
	if err != nil || cmd == nil {
		return err
	}
	return cmd.Scan(value)
}

// GetAnyWithGob 以Gob反序列化形式获取指定值
func (t *cmdString) GetAnyWithGob(key RedisKey, value any, keyAppend ...any) error {
	bytes, err := t.GetBytes(key, keyAppend...)
	if err != nil {
		return err
	}
	return gob.Decode(bytes, value)
}
