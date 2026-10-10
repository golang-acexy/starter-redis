//go:build integration

package redisstarter

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func startRegressionRedis(t *testing.T) *redis.Client {
	t.Helper()
	binary, err := exec.LookPath("redis-server")
	if err != nil {
		t.Skip("redis-server is required for isolated expiration integration tests")
	}
	// Use a private Unix socket so tests neither reuse configured endpoints nor reserve TCP ports.
	directory := t.TempDir()
	socket := filepath.Join(directory, "redis.sock")
	if len(socket) >= 100 {
		// Unix socket path limits can be shorter than the platform's temporary directory paths.
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		address := listener.Addr().String()
		_ = listener.Close()
		_, port, _ := net.SplitHostPort(address)
		return launchRegressionRedis(t, binary, directory, []string{"--bind", "127.0.0.1", "--port", port}, &redis.Options{Addr: address})
	}
	return launchRegressionRedis(t, binary, directory, []string{"--port", "0", "--unixsocket", socket, "--unixsocketperm", "700"}, &redis.Options{Network: "unix", Addr: socket})
}

func launchRegressionRedis(t *testing.T, binary, directory string, addressArgs []string, options *redis.Options) *redis.Client {
	t.Helper()
	args := append(addressArgs, "--save", "", "--appendonly", "no", "--dir", directory)
	process := exec.Command(binary, args...)
	if err := process.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		_ = process.Wait()
		close(done)
	}()
	t.Cleanup(func() {
		_ = process.Process.Kill()
		<-done
	})
	options.DialTimeout = 100 * time.Millisecond
	options.MaxRetries = -1
	client := redis.NewClient(options)
	t.Cleanup(func() { _ = client.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	for {
		if err := client.Ping(ctx).Err(); err == nil {
			return client
		}
		select {
		case <-done:
			t.Fatal("isolated redis-server exited before readiness")
		case <-ctx.Done():
			t.Fatal("isolated redis-server did not become ready")
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func expiryWriters() map[string]func(RedisKey) error {
	return map[string]func(RedisKey) error{
		"hash": func(key RedisKey) error { return HashCmd().HSet(key, "field", "value") },
		"hash batch": func(key RedisKey) error {
			return HashCmd().HMSet(key, map[string]string{"field": "value", "other": "second"})
		},
		"hash gob":       func(key RedisKey) error { return HashCmd().HSetAny(key, "field", 42) },
		"hash batch gob": func(key RedisKey) error { return HashCmd().HMSetAny(key, map[string]any{"field": 42}) },
		"set":            func(key RedisKey) error { return SetCmd().SAdd(key, "member") },
		"set batch":      func(key RedisKey) error { return SetCmd().SAdds(key, []any{"first", "second"}) },
	}
}

func TestInitialExpiryAndPreservation(t *testing.T) {
	client := startRegressionRedis(t)
	useRegressionClient(t, client)
	ctx := context.Background()
	for name, write := range expiryWriters() {
		t.Run(name, func(t *testing.T) {
			key := NewRedisKey("expiry:"+name, 30*time.Second)
			if err := write(key); err != nil {
				t.Fatal(err)
			}
			if ttl := client.PTTL(ctx, key.KeyFormat).Val(); ttl <= 0 || ttl > 30*time.Second {
				t.Fatalf("new key TTL = %v", ttl)
			}
			if err := client.Persist(ctx, key.KeyFormat).Err(); err != nil {
				t.Fatal(err)
			}
			if err := write(key); err != nil {
				t.Fatal(err)
			}
			if ttl := client.PTTL(ctx, key.KeyFormat).Val(); ttl <= 0 || ttl > 30*time.Second {
				t.Fatalf("persistent key did not receive expiration: %v", ttl)
			}
			if err := client.PExpire(ctx, key.KeyFormat, 400*time.Millisecond).Err(); err != nil {
				t.Fatal(err)
			}
			if ttl := client.TTL(ctx, key.KeyFormat).Val(); ttl != 0 {
				t.Fatalf("expected an existing zero-second TTL, got %v", ttl)
			}
			if err := write(key); err != nil {
				t.Fatal(err)
			}
			if ttl := client.PTTL(ctx, key.KeyFormat).Val(); ttl <= 0 || ttl > 400*time.Millisecond {
				t.Fatalf("existing short expiration was refreshed: %v", ttl)
			}
			zeroKey := NewRedisKey("no-expiry:" + name)
			if err := write(zeroKey); err != nil {
				t.Fatal(err)
			}
			if ttl := client.TTL(ctx, zeroKey.KeyFormat).Val(); ttl != -1 {
				t.Fatalf("zero-expiry key TTL = %v", ttl)
			}
			shortKey := NewRedisKey("minimum-expiry:"+name, time.Millisecond)
			if err := write(shortKey); err != nil {
				t.Fatal(err)
			}
			if ttl := client.PTTL(ctx, shortKey.KeyFormat).Val(); ttl <= 400*time.Millisecond || ttl > time.Second {
				t.Fatalf("minimum positive expiration = %v", ttl)
			}
		})
	}
}

func TestExpiryFailuresAreReturned(t *testing.T) {
	client := startRegressionRedis(t)
	useRegressionClient(t, client)
	ctx := context.Background()
	if err := client.Do(ctx, "ACL", "SETUSER", "default", "-expire").Err(); err != nil {
		t.Fatal(err)
	}
	for name, write := range expiryWriters() {
		if err := write(NewRedisKey("denied-expiry:"+name, time.Minute)); err == nil {
			t.Fatalf("%s suppressed the EXPIRE permission error", name)
		}
	}
}

func TestConcurrentInitialExpiryDoesNotRefresh(t *testing.T) {
	client := startRegressionRedis(t)
	useRegressionClient(t, client)
	ctx := context.Background()
	key := NewRedisKey("concurrent-set", 30*time.Second)
	if err := SetCmd().SAdd(key, "seed"); err != nil {
		t.Fatal(err)
	}
	if err := client.PExpire(ctx, key.KeyFormat, 5*time.Second).Err(); err != nil {
		t.Fatal(err)
	}
	results := make(chan error, 20)
	for index := 0; index < cap(results); index++ {
		go func(index int) { results <- SetCmd().SAdd(key, fmt.Sprintf("member-%d", index)) }(index)
	}
	for index := 0; index < cap(results); index++ {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	if ttl := client.PTTL(ctx, key.KeyFormat).Val(); ttl <= 0 || ttl > 5*time.Second {
		t.Fatalf("concurrent writes refreshed existing expiration: %v", ttl)
	}
	if count := client.SCard(ctx, key.KeyFormat).Val(); count != 21 {
		t.Fatalf("member count = %d, want 21", count)
	}
}
