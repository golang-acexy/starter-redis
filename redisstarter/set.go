package redisstarter

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type cmdSet struct {
}

var setCmd = new(cmdSet)

func SetCmd() *cmdSet {
	return setCmd
}

func sAdd(key RedisKey, value []any, keyAppend ...any) error {
	if value == nil {
		return ErrNilValue
	}
	originKey := key.RawKeyString(keyAppend...)
	cmd := rawRedisClient().SAdd(context.Background(), originKey, value...)
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

func sRem(key RedisKey, value []any, keyAppend ...any) *redis.IntCmd {
	originKey := key.RawKeyString(keyAppend...)
	return rawRedisClient().SRem(context.Background(), originKey, value...)
}

// SAdd 增加单个元素
func (*cmdSet) SAdd(key RedisKey, value any, keyAppend ...any) error {
	return sAdd(key, []any{value}, keyAppend...)
}

// SAdds 增加多个元素
func (*cmdSet) SAdds(key RedisKey, value []any, keyAppend ...any) error {
	if len(value) == 0 {
		return ErrNilValue
	}
	slice := make([]any, len(value))
	for i, v := range value {
		slice[i] = v
	}
	return sAdd(key, slice, keyAppend...)
}

// SRem 删除单个元素
func (*cmdSet) SRem(key RedisKey, value any, keyAppend ...any) (int64, error) {
	originKey := key.RawKeyString(keyAppend...)
	cmd := rawRedisClient().SRem(context.Background(), originKey, value)
	if cmd.Err() != nil {
		return 0, cmd.Err()
	}
	return cmd.Val(), nil
}

// SRems 删除多个元素
func (*cmdSet) SRems(key RedisKey, value []any, keyAppend ...any) (int64, error) {
	originKey := key.RawKeyString(keyAppend...)
	cmd := rawRedisClient().SRem(context.Background(), originKey, value...)
	if cmd.Err() != nil {
		return 0, cmd.Err()
	}
	return cmd.Val(), nil
}

// SCard 获取集合元素个数
func (*cmdSet) SCard(key RedisKey, keyAppend ...any) (int64, error) {
	cmd := rawRedisClient().SCard(context.Background(), key.RawKeyString(keyAppend...))
	if cmd.Err() != nil {
		return 0, cmd.Err()
	}
	return cmd.Val(), nil
}

// SMembers 获取集合元素
func (*cmdSet) SMembers(key RedisKey, keyAppend ...any) ([]string, error) {
	cmd := rawRedisClient().SMembers(context.Background(), key.RawKeyString(keyAppend...))
	if cmd.Err() != nil {
		return nil, cmd.Err()
	}
	return cmd.Val(), nil
}

// SMembersScan 获取集合元素
func (*cmdSet) SMembersScan(key RedisKey, value any, keyAppend ...any) error {
	cmd := rawRedisClient().SMembers(context.Background(), key.RawKeyString(keyAppend...))
	if cmd.Err() != nil {
		return cmd.Err()
	}
	return cmd.ScanSlice(value)
}

// SMembersMap 获取集合元素 作用是通过map key去重复
func (*cmdSet) SMembersMap(key RedisKey, keyAppend ...any) (map[string]struct{}, error) {
	cmd := rawRedisClient().SMembersMap(context.Background(), key.RawKeyString(keyAppend...))
	if cmd.Err() != nil {
		return nil, cmd.Err()
	}
	return cmd.Val(), nil
}
