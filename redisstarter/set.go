package redisstarter

import (
	"context"
	"errors"
	"github.com/redis/go-redis/v9"
)

type cmdSet struct {
}

var setCmd = new(cmdSet)

func SetCmd() *cmdSet {
	return setCmd
}

func sAdd(key RedisKey, value []interface{}, keyAppend ...interface{}) error {
	if value == nil {
		return errors.New("nil value")
	}
	originKey := key.RawKeyString(keyAppend...)
	cmd := redisClient.SAdd(context.Background(), originKey, value...)
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

func sRem(key RedisKey, value []interface{}, keyAppend ...interface{}) *redis.IntCmd {
	originKey := key.RawKeyString(keyAppend...)
	return redisClient.SRem(context.Background(), originKey, value...)
}

// SAdd 增加单个元素
func (*cmdSet) SAdd(key RedisKey, value interface{}, keyAppend ...interface{}) error {
	return sAdd(key, []interface{}{value}, keyAppend...)
}

// SAdds 增加多个元素
func (*cmdSet) SAdds(key RedisKey, value []interface{}, keyAppend ...interface{}) error {
	if len(value) == 0 {
		return errors.New("nil value")
	}
	slice := make([]interface{}, len(value))
	for i, v := range value {
		slice[i] = v
	}
	return sAdd(key, slice, keyAppend...)
}

// SRem 删除单个元素
func (*cmdSet) SRem(key RedisKey, value interface{}, keyAppend ...interface{}) (int64, error) {
	originKey := key.RawKeyString(keyAppend...)
	cmd := redisClient.SRem(context.Background(), originKey, value)
	if cmd.Err() != nil {
		return 0, cmd.Err()
	}
	return cmd.Val(), nil
}

// SRems 删除多个元素
func (*cmdSet) SRems(key RedisKey, value []interface{}, keyAppend ...interface{}) (int64, error) {
	originKey := key.RawKeyString(keyAppend...)
	cmd := redisClient.SRem(context.Background(), originKey, value...)
	if cmd.Err() != nil {
		return 0, cmd.Err()
	}
	return cmd.Val(), nil
}

// SCard 获取集合元素个数
func (*cmdSet) SCard(key RedisKey, keyAppend ...interface{}) (int64, error) {
	cmd := redisClient.SCard(context.Background(), key.RawKeyString(keyAppend...))
	if cmd.Err() != nil {
		return 0, cmd.Err()
	}
	return cmd.Val(), nil
}

// SMembers 获取集合元素
func (*cmdSet) SMembers(key RedisKey, keyAppend ...interface{}) ([]string, error) {
	cmd := redisClient.SMembers(context.Background(), key.RawKeyString(keyAppend...))
	if cmd.Err() != nil {
		return nil, cmd.Err()
	}
	return cmd.Val(), nil
}

// SMembersScan 获取集合元素
func (*cmdSet) SMembersScan(key RedisKey, value interface{}, keyAppend ...interface{}) error {
	cmd := redisClient.SMembers(context.Background(), key.RawKeyString(keyAppend...))
	if cmd.Err() != nil {
		return cmd.Err()
	}
	return cmd.ScanSlice(value)
}

// SMembersMap 获取集合元素 作用是通过map key去重复
func (*cmdSet) SMembersMap(key RedisKey, keyAppend ...interface{}) (map[string]struct{}, error) {
	cmd := redisClient.SMembersMap(context.Background(), key.RawKeyString(keyAppend...))
	if cmd.Err() != nil {
		return nil, cmd.Err()
	}
	return cmd.Val(), nil
}
