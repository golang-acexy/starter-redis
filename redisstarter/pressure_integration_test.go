//go:build integration

package redisstarter

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
	"weak"

	"github.com/redis/go-redis/v9"
)

func pressureOptions(t *testing.T) redis.UniversalOptions {
	t.Helper()
	opts := redis.UniversalOptions{PoolSize: 8, MaxActiveConns: 8, MaxRetries: -1, DialTimeout: time.Second, ReadTimeout: time.Second, WriteTimeout: time.Second}
	if addresses := os.Getenv("REDIS_PRESSURE_ADDRS"); addresses != "" {
		opts.Addrs = strings.Split(addresses, ",")
		opts.Username = os.Getenv("REDIS_PRESSURE_USERNAME")
		opts.Password = os.Getenv("REDIS_PRESSURE_PASSWORD")
	} else {
		server := startRegressionRedis(t)
		address, network := server.Options().Addr, server.Options().Network
		opts.Addrs = []string{address}
		opts.Dialer = func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, network, address)
		}
	}
	return opts
}

func pressureStarter(t *testing.T, opts redis.UniversalOptions) (*RedisStarter, redis.UniversalClient) {
	t.Helper()
	starter := &RedisStarter{Config: RedisConfig{UniversalOptions: opts}}
	instance, err := starter.Start()
	if err != nil {
		t.Fatal("pressure Redis startup failed")
	}
	t.Cleanup(func() {
		if RawRedisClient() != nil {
			g, s, err := starter.Stop(5 * time.Second)
			if err != nil || !g || !s {
				t.Error("pressure shutdown failed")
			}
		}
	})
	return starter, instance.(redis.UniversalClient)
}

func waitPressure(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatal("resource did not settle")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestRedisMemoryPressure(t *testing.T) {
	if os.Getenv("REDIS_MEMORY_STRESS") != "1" {
		t.Skip("set REDIS_MEMORY_STRESS=1")
	}
	_, client := pressureStarter(t, pressureOptions(t))
	ctx := context.Background()
	prefix := fmt.Sprintf("starter-pressure:%d:", time.Now().UnixNano())
	keys := make([]RedisKey, 16)
	payload := strings.Repeat("x", 65536)
	for i := range keys {
		keys[i] = NewRedisKey(fmt.Sprintf("%s%d", prefix, i), time.Minute)
		if err := StringCmd().Set(keys[i], payload); err != nil {
			t.Fatal("fixture failed")
		}
	}
	t.Cleanup(func() {
		for _, key := range keys {
			if err := client.Del(ctx, key.KeyFormat).Err(); err != nil {
				t.Error("owned key cleanup failed")
			}
		}
	})
	for _, mode := range []string{"driver_get", "mapper_get", "mapper_set"} {
		t.Run(mode, func(t *testing.T) {
			run := func(count int) {
				var wg sync.WaitGroup
				for w := 0; w < 16; w++ {
					wg.Add(1)
					go func(w int) {
						defer wg.Done()
						for i := w; i < count; i += 16 {
							var value string
							var err error
							switch mode {
							case "driver_get":
								value, err = client.Get(ctx, keys[w].KeyFormat).Result()
							case "mapper_get":
								value, err = StringCmd().Get(keys[w])
							case "mapper_set":
								err = StringCmd().Set(keys[w], payload)
							}
							if err != nil || (mode != "mapper_set" && value != payload) {
								t.Error("command or result failed")
								return
							}
						}
					}(w)
				}
				wg.Wait()
			}
			run(100)
			runtime.GC()
			runtime.GC()
			var baseline runtime.MemStats
			runtime.ReadMemStats(&baseline)
			baseG := runtime.NumGoroutine()
			for batch := 1; batch <= 5; batch++ {
				start := time.Now()
				run(500)
				elapsed := time.Since(start)
				runtime.GC()
				runtime.GC()
				var mem runtime.MemStats
				runtime.ReadMemStats(&mem)
				stats := client.PoolStats()
				if stats.TotalConns != stats.IdleConns {
					t.Error("connections remain borrowed after workers finish")
				}
				t.Logf("batch=%d ops_s=%.0f heap=%d delta=%d allocated=%d goroutines=%d baseline_goroutines=%d total_conns=%d idle_conns=%d", batch, 500/elapsed.Seconds(), mem.HeapAlloc, int64(mem.HeapAlloc)-int64(baseline.HeapAlloc), mem.TotalAlloc-baseline.TotalAlloc, runtime.NumGoroutine(), baseG, stats.TotalConns, stats.IdleConns)
			}
		})
	}
	t.Run("locks", func(t *testing.T) {
		for i := 0; i < 1000; i++ {
			key := NewRedisKey(prefix+"lock", time.Second)
			if err := TryLock(key, func() error { return nil }); err != nil {
				t.Fatal("lock failed")
			}
		}
		expected := errors.New("callback failure")
		key := NewRedisKey(prefix+"lock", time.Second)
		if err := TryLock(key, func() error { return expected }); !errors.Is(err, expected) {
			t.Fatal("callback error lost")
		}
		func() {
			defer func() {
				if recover() == nil {
					t.Error("callback panic lost")
				}
			}()
			_ = TryLock(key, func() error { panic("test panic") })
		}()
		if client.Exists(ctx, key.KeyFormat).Val() != 0 {
			t.Fatal("lease remained after callback")
		}
	})
	t.Run("subscription_churn", func(t *testing.T) {
		refs := make([]weak.Pointer[topicSubscription], 0, 100)
		for i := 0; i < 100; i++ {
			key := NewRedisKey(fmt.Sprintf("%stopic:%d", prefix, i))
			cctx, cancelCtx := context.WithCancel(ctx)
			first, cancelFirst, err := TopicCmd().SubscribeRepeat(cctx, key)
			if err != nil {
				t.Fatal("subscribe failed")
			}
			second, cancelSecond, err := TopicCmd().SubscribeRepeat(cctx, key)
			if err != nil {
				cancelFirst()
				cancelCtx()
				t.Fatal("repeat failed")
			}
			topicCmd.pubSubsMutex.Lock()
			refs = append(refs, weak.Make(topicCmd.pubSubs[key.KeyFormat]))
			topicCmd.pubSubsMutex.Unlock()
			if err := TopicCmd().Publish(key, "message"); err != nil {
				t.Fatal("publish failed")
			}
			for _, ch := range []<-chan *redis.Message{first, second} {
				select {
				case msg := <-ch:
					if msg == nil || msg.Payload != "message" {
						t.Fatal("broadcast failed")
					}
				case <-time.After(3 * time.Second):
					t.Fatal("delivery timeout")
				}
			}
			cancelFirst()
			cancelFirst()
			cancelSecond()
			cancelCtx()
		}
		waitPressure(t, func() bool {
			runtime.GC()
			runtime.GC()
			topicCmd.pubSubsMutex.Lock()
			empty := len(topicCmd.pubSubs) == 0 && len(topicCmd.pending) == 0
			topicCmd.pubSubsMutex.Unlock()
			if !empty {
				return false
			}
			for _, ref := range refs {
				if ref.Value() != nil {
					return false
				}
			}
			return true
		})
	})
}

type pressureClientRefs struct {
	runtime weak.Pointer[redisRuntime]
	single  weak.Pointer[redis.Client]
	cluster weak.Pointer[redis.ClusterClient]
}

func redisPressureCycle(t *testing.T, opts redis.UniversalOptions) pressureClientRefs {
	t.Helper()
	starter := &RedisStarter{Config: RedisConfig{UniversalOptions: opts}}
	if _, err := starter.Start(); err != nil {
		t.Fatal("cycle start failed")
	}
	defer func() {
		if RawRedisClient() != nil {
			starter.Stop(5 * time.Second)
		}
	}()
	ref := pressureClientRefs{runtime: weak.Make(redisRuntimeState.Load())}
	switch client := RawRedisClient().(type) {
	case *redis.Client:
		ref.single = weak.Make(client)
	case *redis.ClusterClient:
		ref.cluster = weak.Make(client)
	default:
		t.Fatal("unexpected client type")
	}
	key := NewRedisKey(fmt.Sprintf("starter-pressure:cycle:%d", time.Now().UnixNano()))
	ch, _, err := TopicCmd().SubscribeRepeat(context.Background(), key)
	if err != nil {
		t.Fatal("cycle subscribe failed")
	}
	g, s, err := starter.Stop(5 * time.Second)
	if err != nil || !g || !s {
		t.Fatal("cycle stop failed")
	}
	select {
	case _, open := <-ch:
		if open {
			t.Fatal("channel not closed")
		}
	case <-time.After(time.Second):
		t.Fatal("channel still blocked")
	}
	return ref
}

func TestRedisPressureLifecycle(t *testing.T) {
	if os.Getenv("REDIS_MEMORY_STRESS") != "1" {
		t.Skip("set REDIS_MEMORY_STRESS=1")
	}
	opts := pressureOptions(t)
	refs := make([]pressureClientRefs, 0, 20)
	for i := 0; i < 20; i++ {
		refs = append(refs, redisPressureCycle(t, opts))
	}
	started := time.Now()
	deadline := time.Now().Add(75 * time.Second)
	for {
		runtime.GC()
		runtime.GC()
		live := 0
		for _, ref := range refs {
			if ref.runtime.Value() != nil || ref.single.Value() != nil || ref.cluster.Value() != nil {
				live++
			}
		}
		if live == 0 {
			t.Logf("all clients reclaimed after %s", time.Since(started))
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("clients still retained after driver cleanup window: %d", live)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// TestBPopCancellationBoundary verifies cancellation without a producer waking the blocked read.
func TestBPopCancellationBoundary(t *testing.T) {
	client := startRegressionRedis(t)
	useRegressionClient(t, client)
	for _, timeout := range []time.Duration{0, 30 * time.Second} {
		for _, right := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/right=%t", timeout, right), func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				ch := ListCmd().BPop(ctx, right, timeout, NewRedisKey("pressure:block"))
				waitPressure(t, func() bool {
					return strings.Contains(client.Info(context.Background(), "clients").Val(), "blocked_clients:1")
				})
				start := time.Now()
				cancel()
				select {
				case _, open := <-ch:
					if open {
						t.Fatal("unexpected item")
					}
				case <-time.After(3 * time.Second):
					t.Fatal("cancellation did not stop blocked stream")
				}
				t.Logf("cancel_to_close=%s", time.Since(start))
				waitPressure(t, func() bool {
					return strings.Contains(client.Info(context.Background(), "clients").Val(), "blocked_clients:0")
				})
			})
		}
	}
}

func TestBPopContinuesAfterPollTimeout(t *testing.T) {
	client := startRegressionRedis(t)
	useRegressionClient(t, client)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	key := NewRedisKey("pressure:poll")
	ch := ListCmd().BPop(ctx, false, 0, key)
	// Observe completed server timeouts rather than relying on a fixed sleep.
	waitPressure(t, func() bool {
		info := client.Info(context.Background(), "commandstats").Val()
		for _, line := range strings.Split(info, "\n") {
			if strings.HasPrefix(line, "cmdstat_blpop:calls=") {
				var calls int
				fmt.Sscanf(line, "cmdstat_blpop:calls=%d", &calls)
				return calls >= 2
			}
		}
		return false
	})
	if err := client.RPush(context.Background(), key.KeyFormat, "first", "second").Err(); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"first", "second"} {
		select {
		case got := <-ch:
			if got != expected {
				t.Fatalf("got %q", got)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("stream stopped after polling timeout")
		}
	}
	cancel()
	select {
	case <-ch:
	case <-time.After(3 * time.Second):
		t.Fatal("stream did not close")
	}
}

func TestBPopCancelUndrainedDelivery(t *testing.T) {
	client := startRegressionRedis(t)
	useRegressionClient(t, client)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	key := NewRedisKey("pressure:delivery")
	if err := client.RPush(context.Background(), key.KeyFormat, "first", "second").Err(); err != nil {
		t.Fatal(err)
	}
	ch := ListCmd().BPop(ctx, true, 0, key)
	waitPressure(t, func() bool { return client.LLen(context.Background(), key.KeyFormat).Val() == 1 })
	cancel()
	deadline := time.After(time.Second)
	for {
		select {
		case value, open := <-ch:
			if !open {
				return
			}
			if value != "second" {
				t.Fatal("right-pop order changed")
			}
		case <-deadline:
			t.Fatal("undrained delivery ignored cancellation")
		}
	}
}

func TestStringSetPreservesContentsAndTTL(t *testing.T) {
	client := startRegressionRedis(t)
	useRegressionClient(t, client)
	ctx := context.Background()
	key := NewRedisKey("pressure:string", time.Minute)
	for _, value := range []string{"", "\x00\xfftext\n", strings.Repeat("x", 65536)} {
		if err := StringCmd().Set(key, value); err != nil {
			t.Fatal(err)
		}
		got, err := client.Get(ctx, key.KeyFormat).Result()
		if err != nil || got != value {
			t.Fatal("stored bytes changed")
		}
		if ttl := client.PTTL(ctx, key.KeyFormat).Val(); ttl <= 0 || ttl > time.Minute {
			t.Fatal("TTL changed")
		}
	}
	if err := StringCmd().Set(NewRedisKey(key.KeyFormat, redis.KeepTTL), "keep"); err != nil {
		t.Fatal(err)
	}
	if client.PTTL(ctx, key.KeyFormat).Val() <= 0 {
		t.Fatal("KeepTTL lost expiration")
	}
	if err := StringCmd().Set(NewRedisKey(key.KeyFormat), "persistent"); err != nil {
		t.Fatal(err)
	}
	if client.TTL(ctx, key.KeyFormat).Val() != -1 {
		t.Fatal("zero expiry did not remove expiration")
	}
}
