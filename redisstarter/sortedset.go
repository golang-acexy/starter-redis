package redisstarter

import (
	"context"
	"github.com/acexy/golang-toolkit/math/conversion"
	"github.com/redis/go-redis/v9"
)

type cmdSortedSet struct {
}

var sortedSetCmd = new(cmdSortedSet)

func SortedSetCmd() *cmdSortedSet {
	return sortedSetCmd
}

// ZAdd 新增单个元素
func (*cmdSortedSet) ZAdd(key RedisKey, member redis.Z, keyAppend ...interface{}) error {
	cmd := redisClient.ZAdd(context.Background(), OriginKeyString(key.KeyFormat, keyAppend...), member)
	return cmd.Err()
}

// ZAdds 新增多个元素
func (*cmdSortedSet) ZAdds(key RedisKey, member []redis.Z, keyAppend ...interface{}) (int64, error) {
	cmd := redisClient.ZAdd(context.Background(), OriginKeyString(key.KeyFormat, keyAppend...), member...)
	if cmd.Err() != nil {
		return 0, cmd.Err()
	}
	return cmd.Val(), nil
}

// ZRem 删除元素
func (*cmdSortedSet) ZRem(key RedisKey, members interface{}, keyAppend ...interface{}) (int64, error) {
	cmd := redisClient.ZRem(context.Background(), OriginKeyString(key.KeyFormat, keyAppend...), members)
	if cmd.Err() != nil {
		return 0, cmd.Err()
	}
	return cmd.Val(), nil
}

// ZRems 删除多个元素
func (*cmdSortedSet) ZRems(key RedisKey, members []interface{}, keyAppend ...interface{}) (int64, error) {
	cmd := redisClient.ZRem(context.Background(), OriginKeyString(key.KeyFormat, keyAppend...), members...)
	if cmd.Err() != nil {
		return 0, cmd.Err()
	}
	return cmd.Val(), nil
}

// ZCount 统计分数在某个范围内的元素个数 含 min, max
func (*cmdSortedSet) ZCount(key RedisKey, min, max float64, keyAppend ...interface{}) (int64, error) {
	cmd := redisClient.ZCount(context.Background(), OriginKeyString(key.KeyFormat, keyAppend...), conversion.FromFloat64(min), conversion.FromFloat64(max))
	if cmd.Err() != nil {
		return 0, cmd.Err()
	}
	return cmd.Val(), nil
}

// ZRange 按排名范围获取元素 从低到高 排名从0开始
func (*cmdSortedSet) ZRange(key RedisKey, start, stop int64, keyAppend ...interface{}) ([]string, error) {
	cmd := redisClient.ZRange(context.Background(), OriginKeyString(key.KeyFormat, keyAppend...), start, stop)
	if cmd.Err() != nil {
		return nil, cmd.Err()
	}
	return cmd.Val(), nil
}

// ZRevRange 按排名范围获取元素 从高到低 排名从0开始
func (*cmdSortedSet) ZRevRange(key RedisKey, start, stop int64, keyAppend ...interface{}) ([]string, error) {
	cmd := redisClient.ZRevRange(context.Background(), OriginKeyString(key.KeyFormat, keyAppend...), start, stop)
	if cmd.Err() != nil {
		return nil, cmd.Err()
	}
	return cmd.Val(), nil
}
