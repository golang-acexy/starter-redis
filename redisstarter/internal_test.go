package redisstarter

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/bsm/redislock"
	"github.com/redis/go-redis/v9"
)

func TestWithHashTagDoesNotModifyInput(t *testing.T) {
	keys := []string{"first", "second"}

	result := withHashTag("group", keys)

	if !reflect.DeepEqual(keys, []string{"first", "second"}) {
		t.Fatalf("输入切片被修改: %v", keys)
	}
	if !reflect.DeepEqual(result, []string{"{group}first", "{group}second"}) {
		t.Fatalf("HashTag 转换结果不正确: %v", result)
	}
}

func TestParseMGetBytesValueRejectsUnexpectedType(t *testing.T) {
	cmd := redis.NewSliceCmd(context.Background())
	cmd.SetVal([]any{"value", int64(1)})

	_, err := parseMGetBytesValue(cmd, nil)
	if !errors.Is(err, ErrNotStringValue) {
		t.Fatalf("期望 ErrNotStringValue，实际为 %v", err)
	}
}

func TestParseMGetStringValuePreservesMissingValue(t *testing.T) {
	cmd := redis.NewSliceCmd(context.Background())
	cmd.SetVal([]any{"value", nil})

	result, err := parseMGetStringValue(cmd, nil)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if !reflect.DeepEqual(result, []string{"value", ""}) {
		t.Fatalf("解析结果不正确: %v", result)
	}
}

func TestTopicCloseAllInvalidatesPendingSubscription(t *testing.T) {
	cmd := &cmdTopic{
		pubSubs: map[string]*redis.PubSub{},
		pending: map[string]struct{}{"topic": {}},
	}

	cmd.closeAll(context.Background())

	if cmd.generation != 1 {
		t.Fatalf("期望订阅代次递增，实际为 %d", cmd.generation)
	}
	if _, ok := cmd.pending["topic"]; !ok {
		t.Fatal("进行中的订阅应由发起方完成清理")
	}
}

func TestRedisRuntimeSnapshotPublication(t *testing.T) {
	previous := redisRuntimeState.Swap(nil)
	defer redisRuntimeState.Store(previous)

	client := redis.NewUniversalClient(&redis.UniversalOptions{Addrs: []string{"127.0.0.1:6379"}})
	defer client.Close()
	runtime := &redisRuntime{client: client, locker: redislock.New(client)}
	redisRuntimeState.Store(runtime)
	if RawRedisClient() != client || RawLockerClient() != runtime.locker {
		t.Fatal("Redis client 与 locker 未从同一运行时快照读取")
	}

	redisRuntimeState.Store(nil)
	if RawRedisClient() != nil || RawLockerClient() != nil {
		t.Fatal("摘除运行时快照后不应继续暴露 Redis 资源")
	}
}
