package redismodule

import (
	"context"
	"fmt"
	"github.com/acexy/golang-toolkit/logger"
	"github.com/bsm/redislock"
	"github.com/golang-acexy/starter-parent/parentmodule/declaration"
	"github.com/redis/go-redis/v9"
	"time"
)

var redisClient redis.UniversalClient
var redisLockerClient *redislock.Client

type RedisKey struct {

	// 最终key值的格式化格式 将使用 fmt.Sprintf(key.KeyFormat, keyAppend) 进行处理
	KeyFormat string
	Expire    time.Duration
}

type RedisModule struct {
	RedisConfig     redis.UniversalOptions
	LazyRedisConfig func() redis.UniversalOptions

	RedisModuleConfig *declaration.ModuleConfig
	RedisInterceptor  func(instance redis.UniversalClient)
}

func (r *RedisModule) ModuleConfig() *declaration.ModuleConfig {
	if r.RedisModuleConfig != nil {
		return r.RedisModuleConfig
	}
	return &declaration.ModuleConfig{
		ModuleName:               "Redis",
		UnregisterAllowAsync:     true,
		UnregisterMaxWaitSeconds: 10,
		UnregisterPriority:       19,
		LoadInterceptor: func(instance interface{}) {
			if r.RedisInterceptor != nil {
				r.RedisInterceptor(instance.(redis.UniversalClient))
			}
		},
	}
}

func (r *RedisModule) Register() (interface{}, error) {
	if r.LazyRedisConfig != nil {
		r.RedisConfig = r.LazyRedisConfig()
	}
	redisClient = redis.NewUniversalClient(&r.RedisConfig)
	status := redisClient.Ping(context.Background())
	err := status.Err()
	if err != nil {
		return nil, err
	}
	redisLockerClient = redislock.New(redisClient)
	logger.Logrus().Traceln(r.ModuleConfig().ModuleName, "started")
	return redisClient, nil
}

func (r *RedisModule) Unregister(maxWaitSeconds uint) (gracefully bool, err error) {
	err = redisClient.Close()
	if err != nil {
		return
	}
	done := make(chan bool)
	go func() {
		for {
			stats := redisClient.PoolStats()
			fmt.Printf("%+v\n", stats)
			if stats.IdleConns == 0 && stats.TotalConns == 0 {
				done <- true
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
	}()
	select {
	case <-done:
		gracefully = true
	case <-time.After(time.Second * time.Duration(maxWaitSeconds)):
		gracefully = false
	}
	return
}

// RawClient 获取原始RedisClient进行操作
func RawClient() redis.UniversalClient {
	return redisClient
}
