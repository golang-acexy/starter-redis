package redisstarter

import (
	"context"
	"time"
)

type cmdKey struct {
}

var keyCmd = new(cmdKey)

// KeyCmd returns shared key helpers. Commands require a running starter and can panic otherwise. Helpers
// support concurrent calls with stable inputs and use background contexts; scalar results can suppress
// driver errors.
func KeyCmd() *cmdKey {
	return keyCmd
}

// Del deletes the formatted key and returns the removed count; driver errors are suppressed and may
// appear as zero.
func (*cmdKey) Del(key RedisKey, keyAppend ...any) int64 {
	return rawRedisClient().Del(context.Background(), key.RawKeyString(keyAppend...)).Val()
}

// MDel deletes raw keys without formatting; driver errors, including invalid empty input, are
// suppressed.
func (*cmdKey) MDel(keys ...string) int64 {
	return rawRedisClient().Del(context.Background(), keys...).Val()
}

// Exists reports whether the formatted key exists; false can also reflect a suppressed driver error.
func (*cmdKey) Exists(key RedisKey, keyAppend ...any) bool {
	return rawRedisClient().Exists(context.Background(), key.RawKeyString(keyAppend...)).Val() > 0
}

// Expire sets the explicit duration, ignoring key.Expire. Nonpositive durations can delete the key;
// false can reflect absence or a suppressed driver error.
func (*cmdKey) Expire(key RedisKey, time time.Duration, keyAppend ...any) bool {
	return rawRedisClient().Expire(context.Background(), key.RawKeyString(keyAppend...), time).Val()
}

// Ttl returns TTL in seconds, -1 for no expiration, -2 for an absent key, or -3 on command error.
func (*cmdKey) Ttl(key RedisKey, keyAppend ...any) float64 {
	cmd := rawRedisClient().TTL(context.Background(), key.RawKeyString(keyAppend...))
	if cmd.Err() != nil {
		return -3
	}
	ttl := cmd.Val()
	if ttl == -1 || ttl == -2 {
		// Preserve Redis sentinel values stored by the driver as nanosecond durations.
		return float64(ttl)
	}
	return ttl.Seconds()
}
