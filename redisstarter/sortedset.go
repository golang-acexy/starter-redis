package redisstarter

import (
	"context"

	"github.com/acexy/golang-toolkit/math/conversion"
	"github.com/redis/go-redis/v9"
)

type cmdSortedSet struct {
}

var sortedSetCmd = new(cmdSortedSet)

// SortedSetCmd returns shared sorted-set helpers. Calls require a running starter and can panic
// otherwise; concurrent calls need stable inputs. Commands use background contexts and ignore
// key.Expire.
func SortedSetCmd() *cmdSortedSet {
	return sortedSetCmd
}

// ZAdd adds or updates one scored member without applying key.Expire.
func (*cmdSortedSet) ZAdd(key RedisKey, member redis.Z, keyAppend ...any) error {
	cmd := rawRedisClient().ZAdd(context.Background(), key.RawKeyString(keyAppend...), member)
	return cmd.Err()
}

// ZAdds adds or updates scored members and returns Redis added count; empty input is passed to Redis and
// may cause a command error.
func (*cmdSortedSet) ZAdds(key RedisKey, member []redis.Z, keyAppend ...any) (int64, error) {
	cmd := rawRedisClient().ZAdd(context.Background(), key.RawKeyString(keyAppend...), member...)
	if cmd.Err() != nil {
		return 0, cmd.Err()
	}
	return cmd.Val(), nil
}

// ZRem removes one member and returns its removal count; absent members return zero.
func (*cmdSortedSet) ZRem(key RedisKey, members any, keyAppend ...any) (int64, error) {
	cmd := rawRedisClient().ZRem(context.Background(), key.RawKeyString(keyAppend...), members)
	if cmd.Err() != nil {
		return 0, cmd.Err()
	}
	return cmd.Val(), nil
}

// ZRems removes multiple members; empty input is passed to Redis and may cause a command error.
func (*cmdSortedSet) ZRems(key RedisKey, members []any, keyAppend ...any) (int64, error) {
	cmd := rawRedisClient().ZRem(context.Background(), key.RawKeyString(keyAppend...), members...)
	if cmd.Err() != nil {
		return 0, cmd.Err()
	}
	return cmd.Val(), nil
}

// ZCount counts members with scores in the inclusive min/max range; numeric bounds are formatted for
// Redis.
func (*cmdSortedSet) ZCount(key RedisKey, min, max float64, keyAppend ...any) (int64, error) {
	cmd := rawRedisClient().ZCount(context.Background(), key.RawKeyString(keyAppend...), conversion.FromFloat64(min), conversion.FromFloat64(max))
	if cmd.Err() != nil {
		return 0, cmd.Err()
	}
	return cmd.Val(), nil
}

// ZRange returns members from low to high by inclusive zero-based rank. Negative ranks count from the
// end; 0,-1 requests all members.
func (*cmdSortedSet) ZRange(key RedisKey, start, stop int64, keyAppend ...any) ([]string, error) {
	cmd := rawRedisClient().ZRange(context.Background(), key.RawKeyString(keyAppend...), start, stop)
	if cmd.Err() != nil {
		return nil, cmd.Err()
	}
	return cmd.Val(), nil
}

// ZRevRange returns members from high to low by inclusive zero-based rank. Negative ranks count from the
// end; 0,-1 requests all members.
func (*cmdSortedSet) ZRevRange(key RedisKey, start, stop int64, keyAppend ...any) ([]string, error) {
	cmd := rawRedisClient().ZRevRange(context.Background(), key.RawKeyString(keyAppend...), start, stop)
	if cmd.Err() != nil {
		return nil, cmd.Err()
	}
	return cmd.Val(), nil
}
