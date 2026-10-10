package redisstarter

import (
	"context"

	"github.com/acexy/golang-toolkit/util/gob"
	"github.com/redis/go-redis/v9"
)

type cmdHash struct {
}

var hashCmd = new(cmdHash)

// HashCmd returns shared hash helpers. Calls require a running starter and can panic otherwise;
// concurrent calls need stable inputs. Commands use background contexts. Positive-expiry writes
// use a Lua script, preserve existing expiration, and return script errors.
func HashCmd() *cmdHash {
	return hashCmd
}

func hSet(key RedisKey, value []any, keyAppend ...any) error {
	if value == nil {
		return ErrNilValue
	}
	originKey := key.RawKeyString(keyAppend...)
	if key.Expire > 0 {
		return writeWithInitialExpiry("HSET", originKey, key.Expire, value)
	}
	cmd := rawRedisClient().HSet(context.Background(), originKey, value)
	if cmd.Err() != nil {
		return cmd.Err()
	}
	return nil
}

func hMSet(key RedisKey, value []any, keyAppend ...any) error {
	if len(value) == 0 {
		return ErrNilValue
	}
	originKey := key.RawKeyString(keyAppend...)
	if key.Expire > 0 {
		return writeWithInitialExpiry("HMSET", originKey, key.Expire, value)
	}
	cmd := rawRedisClient().HMSet(context.Background(), originKey, value)
	if cmd.Err() != nil {
		return cmd.Err()
	}
	return nil
}

func hGet(key RedisKey, name string, keyAppend ...any) *redis.StringCmd {
	return rawRedisClient().HGet(context.Background(), key.RawKeyString(keyAppend...), name)
}

func hMGet(key RedisKey, names []string, keyAppend ...any) *redis.SliceCmd {
	return rawRedisClient().HMGet(context.Background(), key.RawKeyString(keyAppend...), names...)
}

func hGetAll(key RedisKey, keyAppend ...any) *redis.MapStringStringCmd {
	return rawRedisClient().HGetAll(context.Background(), key.RawKeyString(keyAppend...))
}

// HExists reports field existence; false can also reflect a suppressed driver error.
func (*cmdHash) HExists(key RedisKey, name string, keyAppend ...any) bool {
	return rawRedisClient().HExists(context.Background(), key.RawKeyString(keyAppend...), name).Val()
}

// HSet writes one text field. Positive key.Expire is installed only if the key has no expiration,
// atomically with the write using Lua. Existing expiry is preserved; script errors are returned.
// Lua errors do not roll back a write already executed. Expiry uses whole seconds, with a one-second minimum.
func (*cmdHash) HSet(key RedisKey, name, value string, keyAppend ...any) error {
	return hSet(key, []any{name, value}, keyAppend...)
}

// HSetAny Gob-encodes and writes one field with HSet expiration behavior; encoding and data-write errors
// are returned.
func (*cmdHash) HSetAny(key RedisKey, name string, value any, keyAppend ...any) error {
	encode, err := gob.Encode(value)
	if err != nil {
		return err
	}
	return hSet(key, []any{name, encode}, keyAppend...)
}

// HGetAny Gob-decodes one field into a non-nil destination pointer; it preserves redis.Nil and decoding
// errors.
func (*cmdHash) HGetAny(key RedisKey, name string, result any, keyAppend ...any) error {
	cmd := hGet(key, name, keyAppend...)
	if cmd.Err() != nil {
		return cmd.Err()
	}
	bytes, err := cmd.Bytes()
	if err != nil {
		return err
	}
	return gob.Decode(bytes, result)
}

// HMSet writes text fields; nil or empty data returns ErrNilValue. Positive-expiry writes use the
// same atomic, non-refreshing expiration policy and error handling as HSet.
func (*cmdHash) HMSet(key RedisKey, data map[string]string, keyAppend ...any) error {
	array := make([]any, len(data)*2)
	index := 0
	for k, v := range data {
		array[index] = k
		index++
		array[index] = v
		index++
	}
	return hMSet(key, array, keyAppend...)
}

// HMSetAny Gob-encodes all fields before the write; nil or empty data returns ErrNilValue.
// Expiration follows HMSet, and encoding or script errors are returned.
func (*cmdHash) HMSetAny(key RedisKey, data map[string]any, keyAppend ...any) error {
	array := make([]any, len(data)*2)
	index := 0
	for k, v := range data {
		array[index] = k
		index++
		b, err := gob.Encode(v)
		if err != nil {
			return err
		}
		array[index] = b
		index++
	}
	return hMSet(key, array, keyAppend...)
}

// HGet reads one field and preserves redis.Nil for absence.
func (*cmdHash) HGet(key RedisKey, name string, keyAppend ...any) (string, error) {
	cmd := hGet(key, name, keyAppend...)
	if err := cmd.Err(); err != nil {
		return "", err
	}
	return cmd.Val(), nil
}

// HMGet reads fields in names order; absent values become empty strings. Empty names are passed to Redis
// and may cause a command error.
func (*cmdHash) HMGet(key RedisKey, names []string, keyAppend ...any) ([]string, error) {
	cmd := hMGet(key, names, keyAppend...)
	if err := cmd.Err(); err != nil {
		return nil, err
	}
	result, err := cmd.Result()
	if err != nil {
		return nil, err
	}
	m := make([]string, len(result))
	for i, v := range result {
		if v != nil {
			str, ok := v.(string)
			if !ok {
				return nil, ErrNotStringValue
			}
			m[i] = str
		}
	}
	return m, nil
}

// HGetAll returns all fields; an absent key yields an empty result and driver errors are returned.
func (*cmdHash) HGetAll(key RedisKey, keyAppend ...any) (map[string]string, error) {
	cmd := hGetAll(key, keyAppend...)
	if err := cmd.Err(); err != nil {
		return nil, err
	}
	return cmd.Result()
}
