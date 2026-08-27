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

type RedisKey struct {

	// 最终key值的格式化格式 将使用 fmt.Sprintf(key.KeyFormat, keyAppend) 进行处理
	KeyFormat string
	// Key 过期时间(如果可以设置) 当该RedisKey作用与Locker时，该时间指明了自动释放锁时间 如果为零值则会立即失败
	Expire time.Duration
}

// NewRedisKey 创建一个RedisKey
// keyFormat 最终key值的格式化格式 将使用 fmt.Sprintf(key.KeyFormat, keyAppend) 进行处理
func NewRedisKey(keyFormat string, expire ...time.Duration) RedisKey {
	key := RedisKey{
		KeyFormat: keyFormat,
	}
	if len(expire) > 0 {
		key.Expire = expire[0]
	}
	return key
}

// RawKeyString 获取原始key字符串
func (r RedisKey) RawKeyString(keyAppend ...any) string {
	if len(keyAppend) > 0 {
		return fmt.Sprintf(r.KeyFormat, keyAppend...)
	}
	return r.KeyFormat
}

type RedisConfig struct {
	redis.UniversalOptions
	InitFunc func(instance redis.UniversalClient)
}

type RedisStarter struct {
	Config       RedisConfig
	LazyConfig   func() RedisConfig
	config       *RedisConfig
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

// RawRedisClient 获取原始RedisClient进行操作
func RawRedisClient() redis.UniversalClient {
	runtime := redisRuntimeState.Load()
	if runtime == nil {
		return nil
	}
	return runtime.client
}

// RawLockerClient 获取原始RedisLockerClient进行操作
func RawLockerClient() *redislock.Client {
	return rawLockerClient()
}

func rawRedisClient() redis.UniversalClient {
	return RawRedisClient()
}
