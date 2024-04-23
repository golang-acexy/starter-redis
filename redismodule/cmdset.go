package redismodule

import (
	"context"
	"errors"
	"github.com/acexy/golang-toolkit/util/json"
	"github.com/redis/go-redis/v9"
)

type cmdSet struct {
}

var setCmd = new(cmdSet)

func SetCmd() *cmdSet {
	return setCmd
}

func sAdd(ctx context.Context, key RedisKey, value []interface{}, keyAppend ...interface{}) error {
	if value == nil {
		return errors.New("nil value")
	}
	originKey := OriginKeyString(key.KeyFormat, keyAppend...)
	cmd := redisClient.SAdd(ctx, originKey, value...)
	if cmd.Err() != nil {
		return cmd.Err()
	}
	if key.Expire > 0 {
		keyCmd.Expire(ctx, originKey, key.Expire)
	}
	return nil
}

func sMembers(ctx context.Context, key RedisKey, keyAppend ...interface{}) (*redis.StringSliceCmd, error) {
	cmd := redisClient.SMembers(ctx, OriginKeyString(key.KeyFormat, keyAppend...))
	if cmd.Err() != nil {
		if errors.Is(cmd.Err(), redis.Nil) {
			return cmd, nil // wrap nil error
		}
		return nil, cmd.Err()
	}
	return cmd, nil
}

func (*cmdSet) SAdd(ctx context.Context, key RedisKey, value []string, keyAppend ...interface{}) error {
	if len(value) == 0 {
		return errors.New("nil value")
	}
	slice := make([]interface{}, len(value))
	for i, v := range value {
		slice[i] = v
	}
	return sAdd(ctx, key, slice, keyAppend...)
}

func (*cmdSet) SAddBytes(ctx context.Context, key RedisKey, value []byte, keyAppend ...interface{}) error {
	if len(value) == 0 {
		return errors.New("nil value")
	}
	slice := make([]interface{}, len(value))
	for i, v := range value {
		b, e := json.ToJsonBytesError(v)
		if e != nil {
			return e
		}
		slice[i] = b
	}
	return sAdd(ctx, key, slice, keyAppend...)
}

func (*cmdSet) SCard(ctx context.Context, key RedisKey, keyAppend ...interface{}) (int64, error) {
	cmd := redisClient.SCard(ctx, OriginKeyString(key.KeyFormat, keyAppend...))
	if cmd.Err() != nil {
		if errors.Is(cmd.Err(), redis.Nil) {
			return 0, nil // wrap nil error
		}
		return 0, cmd.Err()
	}
	return cmd.Val(), nil
}

func (*cmdSet) SMembers(ctx context.Context, key RedisKey, keyAppend ...interface{}) ([]string, error) {
	cmd, err := sMembers(ctx, key, keyAppend...)
	if err != nil {
		return nil, err
	}
	return cmd.Val(), nil
}

func (*cmdSet) SMembersBytes(ctx context.Context, key RedisKey, keyAppend ...interface{}) ([]*byte, error) {
	cmd, err := sMembers(ctx, key, keyAppend...)
	if err != nil {
		return nil, err
	}
	bytes := new([]*byte)
	err = cmd.ScanSlice(bytes)
	if err != nil {
		return nil, err
	}
	return *bytes, nil
}
