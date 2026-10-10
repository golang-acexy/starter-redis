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

// StringCmd returns shared string helpers. Calls require a running starter and can panic otherwise;
// concurrent calls need stable caller-owned inputs. Commands use background contexts, and batch writes
// do not set expiration.
func StringCmd() *cmdString {
	return stringCmd
}

func set(key RedisKey, value any, keyAppend ...any) error {
	if value == nil {
		return ErrNilValue
	}
	status := rawRedisClient().Set(context.Background(), key.RawKeyString(keyAppend...), value, key.Expire)
	err := status.Err()
	if err != nil {
		return err
	}
	return nil
}

func mset(data []any) error {
	status := rawRedisClient().MSet(context.Background(), data)
	err := status.Err()
	if err != nil {
		return err
	}
	return nil
}

func get(key RedisKey, keyAppend ...any) (*redis.StringCmd, error) {
	cmd := rawRedisClient().Get(context.Background(), key.RawKeyString(keyAppend...))
	if cmd.Err() != nil {
		return nil, cmd.Err()
	}
	return cmd, nil
}

func mget(keys ...string) (*redis.SliceCmd, error) {
	slice := rawRedisClient().MGet(context.Background(), keys...)
	err := slice.Err()
	if err != nil {
		return nil, err
	}
	return slice, nil
}

// Set stores text using key.Expire; zero writes without expiration and redis.KeepTTL retains an existing
// TTL. Empty text is allowed and driver errors are returned.
func (*cmdString) Set(key RedisKey, value string, keyAppend ...any) error {
	return set(key, []byte(value), keyAppend...)
}

// SetBytes stores bytes using key.Expire; a typed nil byte slice is passed to the driver. Do not mutate
// bytes during the call.
func (*cmdString) SetBytes(key RedisKey, value []byte, keyAppend ...any) error {
	return set(key, value, keyAppend...)
}

// SetAny stores a driver-supported scalar or BinaryMarshaler value using key.Expire. A nil interface
// returns ErrNilValue; complex values require supported encoding.
func (*cmdString) SetAny(key RedisKey, value any, keyAppend ...any) error {
	return set(key, value, keyAppend...)
}

// SetAnyWithGob Gob-encodes the value before storing with key.Expire; encoding and driver errors are
// returned.
func (*cmdString) SetAnyWithGob(key RedisKey, value any, keyAppend ...any) error {
	bytes, err := gob.Encode(value)
	if err != nil {
		return err
	}
	return set(key, bytes, keyAppend...)
}

// MSet stores raw key/text pairs without TTL; nil or empty input returns ErrNilValue. Cluster routing
// and cross-slot behavior belong to the driver.
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

// MSetWithHashTag stores pairs using keys prefixed with {hashTag}, without TTL. Empty input returns
// ErrNilValue; hashTag is not validated, so use a non-empty tag without braces for predictable slot
// grouping.
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

// MSetBytes stores raw key/byte pairs without TTL; nil or empty input returns ErrNilValue. Keep map and
// buffers unchanged during use.
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

// MSetBytesWithHashTag stores byte pairs using keys prefixed with {hashTag}, without TTL. Empty input
// returns ErrNilValue; tag contents are not validated and buffers remain caller-owned.
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

// Get reads text and returns redis.Nil for an absent key; other driver errors are preserved.
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

// MGet reads raw keys in input order; missing values become empty strings and cannot be distinguished
// from stored empty text. Empty input returns ErrNilKeys.
func (*cmdString) MGet(keys ...string) ([]string, error) {
	if len(keys) == 0 {
		return nil, ErrNilKeys
	}
	return parseMGetStringValue(mget(keys...))
}

// MGetWithHashTag reads keys prefixed with {hashTag} in input order; missing values become empty
// strings. Empty input returns ErrNilKeys and tag contents are not validated.
func (*cmdString) MGetWithHashTag(hashTag string, keys ...string) ([]string, error) {
	if len(keys) == 0 {
		return nil, ErrNilKeys
	}
	taggedKeys := withHashTag(hashTag, keys)
	return parseMGetStringValue(mget(taggedKeys...))
}

// MGetBytes reads raw keys in input order; absent values remain nil byte slices. Empty input returns
// ErrNilKeys.
func (*cmdString) MGetBytes(keys ...string) ([][]byte, error) {
	if len(keys) == 0 {
		return nil, ErrNilKeys
	}
	return parseMGetBytesValue(mget(keys...))
}

// MGetBytesWithHashTag reads keys prefixed with {hashTag}; absent values remain nil byte slices. Empty
// input returns ErrNilKeys and input keys are not modified.
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

// GetBytes reads bytes and preserves redis.Nil for an absent key; returned data belongs to the caller.
func (*cmdString) GetBytes(key RedisKey, keyAppend ...any) ([]byte, error) {
	cmd, err := get(key, keyAppend...)
	if err != nil || cmd == nil {
		return nil, err
	}
	return cmd.Bytes()
}

// GetAny scans a driver-encoded value into a non-nil supported destination pointer. It preserves
// missing-key and scan errors; Gob values need GetAnyWithGob.
func (*cmdString) GetAny(key RedisKey, value any, keyAppend ...any) error {
	cmd, err := get(key, keyAppend...)
	if err != nil || cmd == nil {
		return err
	}
	return cmd.Scan(value)
}

// GetAnyWithGob decodes a Gob value into the supplied non-nil destination pointer; read and decoding
// errors are returned.
func (t *cmdString) GetAnyWithGob(key RedisKey, value any, keyAppend ...any) error {
	bytes, err := t.GetBytes(key, keyAppend...)
	if err != nil {
		return err
	}
	return gob.Decode(bytes, value)
}
