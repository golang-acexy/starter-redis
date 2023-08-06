package redismodule

import (
	"context"
	"github.com/golang-acexy/starter-parent/parentmodule/declaration"
	"github.com/redis/go-redis/v9"
)

var client *redisClient

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
		UnregisterMaxWaitSeconds: 15,
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
	return nil
}

func (r *RedisModule) Interceptor() *func(instance interface{}) {
	if r.RedisInterceptor != nil {
		return r.RedisInterceptor
	}
	return nil
}

func (r *RedisModule) Unregister(maxWaitSeconds uint) (gracefully bool, err error) {
	return true, nil
}
