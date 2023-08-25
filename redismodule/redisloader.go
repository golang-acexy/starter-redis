package redismodule

import (
	"context"
	"fmt"
	"github.com/acexy/golang-toolkit/log"
	"github.com/bsm/redislock"
	"github.com/golang-acexy/starter-parent/parentmodule/declaration"
	"github.com/redis/go-redis/v9"
	"time"
)

var redisClient redis.UniversalClient
var redisLockerClient *redislock.Client

type RedisModule struct {
	RedisConfig *redis.UniversalOptions

	RedisModuleConfig *declaration.ModuleConfig
	RedisInterceptor  *func(instance interface{})
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
	}
}

func (r *RedisModule) Register(interceptor *func(instance interface{})) error {
	c := redis.NewUniversalClient(r.RedisConfig)
	status := c.Ping(context.Background())
	err := status.Err()
	if err != nil {
		return err
	}
	redisClient = c
	redisLockerClient = redislock.New(redisClient)
	if interceptor != nil {
		(*interceptor)(redisClient)
	}
	log.Logrus().Traceln(r.ModuleConfig().ModuleName, "started")
	return nil
}

// Interceptor instance redis.UniversalClient
func (r *RedisModule) Interceptor() *func(instance interface{}) {
	if r.RedisInterceptor != nil {
		return r.RedisInterceptor
	}
	return nil
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
