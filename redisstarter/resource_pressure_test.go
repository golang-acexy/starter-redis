package redisstarter

import (
	"context"
	"strings"
	"testing"

	"github.com/redis/go-redis/v9"
)

type pressureSetHook struct{}

func (pressureSetHook) DialHook(next redis.DialHook) redis.DialHook { return next }
func (pressureSetHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}
func (pressureSetHook) ProcessHook(redis.ProcessHook) redis.ProcessHook {
	return func(_ context.Context, cmd redis.Cmder) error { cmd.(*redis.StatusCmd).SetVal("OK"); return nil }
}

// BenchmarkStringSetConstruction isolates wrapper allocation from network and server latency.
func BenchmarkStringSetConstruction(b *testing.B) {
	client := redis.NewClient(&redis.Options{Addr: "unused:6379"})
	client.AddHook(pressureSetHook{})
	previous := redisRuntimeState.Swap(&redisRuntime{client: client})
	b.Cleanup(func() { redisRuntimeState.Store(previous); client.Close() })
	payload := strings.Repeat("x", 65536)
	key := NewRedisKey("pressure")
	for _, mode := range []string{"driver", "wrapper"} {
		b.Run(mode, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				var err error
				if mode == "driver" {
					err = client.Set(context.Background(), key.KeyFormat, payload, 0).Err()
				} else {
					err = StringCmd().Set(key, payload)
				}
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkTopicFanout(b *testing.B) {
	for _, count := range []int{1, 100} {
		b.Run(fmtPressureCount(count), func(b *testing.B) {
			subscription := &topicSubscription{subscribers: make(map[uint64]*topicSubscriber)}
			manager := &cmdTopic{pubSubs: map[string]*topicSubscription{"topic": subscription}}
			for i := 0; i < count; i++ {
				subscription.subscribers[uint64(i)] = &topicSubscriber{messages: make(chan *redis.Message, 1), done: make(chan struct{})}
			}
			message := &redis.Message{Payload: "data"}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				manager.deliverMessage("topic", subscription, message)
				for _, s := range subscription.subscribers {
					<-s.messages
				}
			}
		})
	}
}

func fmtPressureCount(count int) string {
	if count == 1 {
		return "one"
	}
	return "hundred"
}
