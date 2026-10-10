package redisstarter

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type cmdSet struct {
}

var setCmd = new(cmdSet)

// SetCmd returns shared set helpers. Calls require a running starter and can panic otherwise; concurrent
// calls need stable inputs. Commands use background contexts; positive-expiry
// writes use Lua, preserve existing expiration, and return script errors.
func SetCmd() *cmdSet {
	return setCmd
}

func sAdd(key RedisKey, value []any, keyAppend ...any) error {
	if value == nil {
		return ErrNilValue
	}
	originKey := key.RawKeyString(keyAppend...)
	if key.Expire > 0 {
		return writeWithInitialExpiry("SADD", originKey, key.Expire, value)
	}
	cmd := rawRedisClient().SAdd(context.Background(), originKey, value...)
	if cmd.Err() != nil {
		return cmd.Err()
	}
	return nil
}

func sRem(key RedisKey, value []any, keyAppend ...any) *redis.IntCmd {
	originKey := key.RawKeyString(keyAppend...)
	return rawRedisClient().SRem(context.Background(), originKey, value...)
}

// SAdd adds one member; duplicates are harmless. Positive key.Expire is installed only when the key
// has no expiration, atomically with the write using Lua. Existing expiry is preserved and errors
// are returned; Lua errors do not roll back writes. Expiry has whole-second precision and a one-second minimum.
func (*cmdSet) SAdd(key RedisKey, value any, keyAppend ...any) error {
	return sAdd(key, []any{value}, keyAppend...)
}

// SAdds adds members; nil or empty input returns ErrNilValue. Expiration follows SAdd,
// and input must stay unchanged during use.
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

// SRem removes one member and returns the removal count; absent members return zero.
func (*cmdSet) SRem(key RedisKey, value any, keyAppend ...any) (int64, error) {
	originKey := key.RawKeyString(keyAppend...)
	cmd := rawRedisClient().SRem(context.Background(), originKey, value)
	if cmd.Err() != nil {
		return 0, cmd.Err()
	}
	return cmd.Val(), nil
}

// SRems removes multiple members; empty input is passed to Redis and may cause a command error.
func (*cmdSet) SRems(key RedisKey, value []any, keyAppend ...any) (int64, error) {
	originKey := key.RawKeyString(keyAppend...)
	cmd := rawRedisClient().SRem(context.Background(), originKey, value...)
	if cmd.Err() != nil {
		return 0, cmd.Err()
	}
	return cmd.Val(), nil
}

// SCard returns member count, including zero for an absent key; driver errors are returned.
func (*cmdSet) SCard(key RedisKey, keyAppend ...any) (int64, error) {
	cmd := rawRedisClient().SCard(context.Background(), key.RawKeyString(keyAppend...))
	if cmd.Err() != nil {
		return 0, cmd.Err()
	}
	return cmd.Val(), nil
}

// SMembers returns all members in unspecified order; an absent key yields an empty result.
func (*cmdSet) SMembers(key RedisKey, keyAppend ...any) ([]string, error) {
	cmd := rawRedisClient().SMembers(context.Background(), key.RawKeyString(keyAppend...))
	if cmd.Err() != nil {
		return nil, cmd.Err()
	}
	return cmd.Val(), nil
}

// SMembersScan fetches all members and scans them into a non-nil supported slice pointer. It does not
// perform cursor-based Redis SCAN.
func (*cmdSet) SMembersScan(key RedisKey, value any, keyAppend ...any) error {
	cmd := rawRedisClient().SMembers(context.Background(), key.RawKeyString(keyAppend...))
	if cmd.Err() != nil {
		return cmd.Err()
	}
	return cmd.ScanSlice(value)
}

// SMembersMap returns members as map keys; an absent key yields an empty result and driver errors are
// preserved.
func (*cmdSet) SMembersMap(key RedisKey, keyAppend ...any) (map[string]struct{}, error) {
	cmd := rawRedisClient().SMembersMap(context.Background(), key.RawKeyString(keyAppend...))
	if cmd.Err() != nil {
		return nil, cmd.Err()
	}
	return cmd.Val(), nil
}
