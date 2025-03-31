package redisstarter

import (
	"context"
	"fmt"
	"github.com/bsm/redislock"
	"github.com/golang-acexy/starter-parent/parent"
	"github.com/redis/go-redis/v9"
	"time"
)

var redisClient redis.UniversalClient
var redisLockerClient *redislock.Client

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
func (r RedisKey) RawKeyString(keyAppend ...interface{}) string {
	if len(keyAppend) > 0 {
		return fmt.Sprintf(r.KeyFormat, keyAppend...)
	}
	return r.KeyFormat
}

type RedisStarter struct {
	RedisConfig     redis.UniversalOptions
	LazyRedisConfig func() redis.UniversalOptions

	InitFunc     func(instance redis.UniversalClient)
	RedisSetting *parent.Setting
}

func (r *RedisStarter) Setting() *parent.Setting {
	if r.RedisSetting != nil {
		return r.RedisSetting
	}
	return parent.NewSetting("Redis-Starter", 19, true, time.Second*10, func(instance interface{}) {
		if r.InitFunc != nil {
			r.InitFunc(instance.(redis.UniversalClient))
		}
	})
}

func (r *RedisStarter) ping() error {
	if redisClient == nil {
		return nil
	}
	return redisClient.Ping(context.Background()).Err()
}
func (r *RedisStarter) closedAllConn() bool {
	if redisClient == nil {
		return true
	}
	stats := redisClient.PoolStats()
	if stats.IdleConns == 0 && stats.TotalConns == 0 {
		return true
	}
	return false
}

func (r *RedisStarter) Start() (interface{}, error) {
	if r.LazyRedisConfig != nil {
		r.RedisConfig = r.LazyRedisConfig()
	}
	redisClient = redis.NewUniversalClient(&r.RedisConfig)
	if err := r.ping(); err != nil {
		return nil, err
	}
	redisLockerClient = redislock.New(redisClient)
	return redisClient, nil
}

func (r *RedisStarter) Stop(maxWaitTime time.Duration) (gracefully, stopped bool, err error) {
	err = redisClient.Close()
	if err != nil {
		if pingErr := r.ping(); pingErr != nil {
			stopped = true
		}
		return
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	go func() {
		for {
			if r.closedAllConn() {
				cancelFunc()
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
	}()
	select {
	case <-ctx.Done():
		gracefully = true
		stopped = r.ping() != nil
	case <-time.After(maxWaitTime):
		gracefully = false
		stopped = r.ping() != nil
	}
	return
}

// RawRedisClient 获取原始RedisClient进行操作
func RawRedisClient() redis.UniversalClient {
	return redisClient
}

// RawLockerClient 获取原始RedisLockerClient进行操作
func RawLockerClient() *redislock.Client {
	return redisLockerClient
}
