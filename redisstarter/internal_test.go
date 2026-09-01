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
	pending := &topicPending{ready: make(chan struct{})}
	cmd := &cmdTopic{
		pubSubs: make(map[string]*topicSubscription),
		pending: map[string]*topicPending{"topic": pending},
	}

	cmd.closeAll(context.Background())

	if cmd.generation != 1 {
		t.Fatalf("期望订阅代次递增，实际为 %d", cmd.generation)
	}
	select {
	case <-pending.ready:
	default:
		t.Fatal("关闭 Starter 时应唤醒等待底层订阅的重复订阅者")
	}
	if !errors.Is(pending.err, ErrSubscriptionClosed) {
		t.Fatalf("期望 ErrSubscriptionClosed，实际为 %v", pending.err)
	}
}

func TestTopicSubscribeReturnsErrorWhenRedisIsNotStarted(t *testing.T) {
	previous := redisRuntimeState.Swap(nil)
	defer redisRuntimeState.Store(previous)

	cmd := &cmdTopic{
		pubSubs: make(map[string]*topicSubscription),
		pending: make(map[string]*topicPending),
	}
	key := NewRedisKey("topic")

	_, err := cmd.Subscribe(context.Background(), key)
	if !errors.Is(err, ErrRedisClientNotStarted) {
		t.Fatalf("期望 ErrRedisClientNotStarted，实际为 %v", err)
	}
	if _, ok := cmd.pending[key.RawKeyString()]; ok {
		t.Fatal("订阅失败后不应保留 pending 标记")
	}
}

func TestSubscribeRepeatUsesIndependentLocalChannels(t *testing.T) {
	key := NewRedisKey("topic")
	baseSubscriber := &topicSubscriber{
		messages: make(chan *redis.Message, topicSubscriberChannelSize),
		done:     make(chan struct{}),
	}
	subscription := &topicSubscription{
		subscribers: map[uint64]*topicSubscriber{1: baseSubscriber},
	}
	cmd := &cmdTopic{
		pubSubs:          map[string]*topicSubscription{key.RawKeyString(): subscription},
		pending:          make(map[string]*topicPending),
		nextSubscriberID: 1,
	}

	ctx, cancelContext := context.WithCancel(context.Background())
	defer cancelContext()
	messages, cancel, err := cmd.SubscribeRepeat(ctx, key)
	if err != nil {
		t.Fatalf("重复订阅失败: %v", err)
	}

	message := &redis.Message{Payload: "message"}
	if !cmd.deliverMessage(key.RawKeyString(), subscription, message) {
		t.Fatal("活动订阅应允许分发消息")
	}
	for _, channel := range []<-chan *redis.Message{baseSubscriber.messages, messages} {
		select {
		case received := <-channel:
			if received != message {
				t.Fatalf("收到意外消息: %#v", received)
			}
		default:
			t.Fatal("每个本地订阅者都应收到消息")
		}
	}

	cancel()
	_, open := <-messages
	if open {
		t.Fatal("取消重复订阅后，其消息通道应关闭")
	}
	if len(subscription.subscribers) != 1 {
		t.Fatalf("取消重复订阅不应影响其他订阅者，实际数量为 %d", len(subscription.subscribers))
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
