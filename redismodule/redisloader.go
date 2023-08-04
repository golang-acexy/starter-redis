package redismodule

import (
	"github.com/golang-acexy/starter-parent/parentmodule/declaration"
	"github.com/redis/go-redis/v9"
)

type RedisModule struct {
	RedisConfig       RedisConfig
	RedisModuleConfig *declaration.ModuleConfig
	RedisInterceptor  *func(instance interface{})
}

// RedisConfig 加载优先级依次向下递减
type RedisConfig struct {
	Cluster    *redis.ClusterOptions
	Standalone *redis.Options
}

func (r *RedisModule) ModuleConfig() *declaration.ModuleConfig {
	return &declaration.ModuleConfig{
		ModuleName:               "Redis",
		UnregisterAllowAsync:     true,
		UnregisterMaxWaitSeconds: 15,
		UnregisterPriority:       19,
	}
}

func (r *RedisModule) Register(interceptor *func(instance interface{})) error {
	if r.RedisConfig.Cluster != nil {
		redis.NewClusterClient(r.RedisConfig.Cluster)
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
