package redisstarter

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bsm/redislock"
	"github.com/golang-acexy/starter-parent/parent"
	"github.com/redis/go-redis/v9"
)

var redisRuntimeState atomic.Pointer[redisRuntime]
var redisLifecycleLock sync.Mutex
var redisState redisLifecycleState

type redisRuntime struct {
	client redis.UniversalClient
	locker *redislock.Client
}

type redisLifecycleState uint8

const (
	redisStopped redisLifecycleState = iota
	redisStarting
	redisRunning
	redisStopping
)

// RedisKey combines key formatting and optional expiration. Treat shared definitions as read-only.
// Expire is command-specific, not a universal sliding TTL; locks use it as a lease duration.
type RedisKey struct {

	// KeyFormat is used by fmt.Sprintf when arguments are present; no arguments return it verbatim.
	KeyFormat string
	// Expire is command-specific. String writes support redis.KeepTTL; locks require a positive lease and
	// the driver truncates it to milliseconds.
	Expire time.Duration
}

// NewRedisKey creates a key definition; only the first optional expiry is used, with zero as the
// default. It does not validate formatting or duration.
func NewRedisKey(keyFormat string, expire ...time.Duration) RedisKey {
	key := RedisKey{
		KeyFormat: keyFormat,
	}
	if len(expire) > 0 {
		key.Expire = expire[0]
	}
	return key
}

// RawKeyString returns KeyFormat unchanged without arguments, otherwise applies fmt.Sprintf without
// placeholder validation.
func (r RedisKey) RawKeyString(keyAppend ...any) string {
	if len(keyAppend) > 0 {
		return fmt.Sprintf(r.KeyFormat, keyAppend...)
	}
	return r.KeyFormat
}

// RedisConfig supplies driver connection settings and a parent-managed initialization callback.
type RedisConfig struct {
	// UniversalOptions selects standalone, Sentinel, or Cluster behavior; nested option data is not deeply
	// copied.
	redis.UniversalOptions
	// InitFunc runs through the default parent settings after startup; direct Start or custom RedisSetting
	// does not invoke it automatically.
	InitFunc func(instance redis.UniversalClient)
}

// RedisStarter manages one process-wide Redis client and lock client.
// Configure before use and serialize configuration resolution; its cached config
// is not protected against concurrent first reads or later mutations. Normal use
// is one startup and one shutdown; default parent settings reject restart.
type RedisStarter struct {
	// Config supplies static options when LazyConfig is nil.
	Config RedisConfig
	// LazyConfig overrides Config and is evaluated once at first resolution, which may occur during Setting
	// before Start.
	LazyConfig func() RedisConfig
	config     *RedisConfig
	// RedisSetting overrides all parent lifecycle settings, including the default InitFunc callback.
	RedisSetting *parent.Setting
}

func (r *RedisStarter) getConfig() *RedisConfig {
	if r.config == nil {
		var config RedisConfig
		if r.LazyConfig != nil {
			config = r.LazyConfig()
		} else {
			config = r.Config
		}
		r.config = &config
	}
	return r.config
}

// Setting returns custom settings or defaults that disable restart, stop asynchronously at priority 19,
// and wait ten seconds. Default resolution caches config and delegates InitFunc to the parent callback;
// custom settings replace that callback.
func (r *RedisStarter) Setting() *parent.Setting {
	if r.RedisSetting != nil {
		return r.RedisSetting
	}
	config := r.getConfig()
	return parent.NewSetting("Redis-Starter", false, 19, true, time.Second*10, func(instance any) {
		if config.InitFunc != nil {
			config.InitFunc(instance.(redis.UniversalClient))
		}
	})
}

// Start creates and pings the process-wide client, then returns the borrowed redis.UniversalClient.
// Failed health checks close the candidate. Ping uses a background context and driver timeouts; direct
// Start does not invoke InitFunc. Configure before use and keep one lifecycle owner.
func (r *RedisStarter) Start() (any, error) {
	redisLifecycleLock.Lock()
	if redisState != redisStopped {
		runtime := redisRuntimeState.Load()
		redisLifecycleLock.Unlock()
		if runtime != nil {
			return runtime.client, ErrRedisStarterAlreadyStarted
		}
		return nil, ErrRedisStarterAlreadyStarted
	}
	redisState = redisStarting
	redisLifecycleLock.Unlock()
	started := false
	defer func() {
		if !started {
			redisLifecycleLock.Lock()
			redisState = redisStopped
			redisLifecycleLock.Unlock()
		}
	}()

	config := r.getConfig()
	client := redis.NewUniversalClient(&config.UniversalOptions)
	if err := client.Ping(context.Background()).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}
	runtime := &redisRuntime{client: client, locker: redislock.New(client)}
	redisLifecycleLock.Lock()
	redisRuntimeState.Store(runtime)
	redisState = redisRunning
	redisLifecycleLock.Unlock()
	started = true
	return client, nil
}

// Stop withdraws the runtime, cancels topics, and closes the shared client. Positive maxWaitTime bounds
// topic cleanup only; nonpositive values use no context deadline. Client.Close is not forcibly timed
// out. It returns stopped=true even on close error; inactive runtime returns ErrRedisClientNotStarted.
func (r *RedisStarter) Stop(maxWaitTime time.Duration) (gracefully, stopped bool, err error) {
	redisLifecycleLock.Lock()
	runtime := redisRuntimeState.Load()
	if redisState != redisRunning || runtime == nil {
		redisLifecycleLock.Unlock()
		return false, true, ErrRedisClientNotStarted
	}
	redisRuntimeState.Store(nil)
	redisState = redisStopping
	redisLifecycleLock.Unlock()
	client := runtime.client
	ctx := context.Background()
	if maxWaitTime > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, maxWaitTime)
		defer cancel()
	}
	topicCmd.closeAll(ctx)
	err = client.Close()
	redisLifecycleLock.Lock()
	redisState = redisStopped
	redisLifecycleLock.Unlock()
	return err == nil, true, err
}

// RawRedisClient returns the starter-owned client or nil when unavailable. Lookup is synchronized, but
// the borrowed client can close afterward; do not close it or use it beyond the starter lifetime.
func RawRedisClient() redis.UniversalClient {
	runtime := redisRuntimeState.Load()
	if runtime == nil {
		return nil
	}
	return runtime.client
}

// RawLockerClient returns the starter-owned lock client or nil when unavailable. Do not treat lookup as
// a lifetime guarantee; individual acquired leases remain caller-owned.
func RawLockerClient() *redislock.Client {
	return rawLockerClient()
}

func rawRedisClient() redis.UniversalClient {
	return RawRedisClient()
}
