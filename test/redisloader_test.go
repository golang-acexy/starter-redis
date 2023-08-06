package test

import (
	"github.com/golang-acexy/starter-parent/parentmodule/declaration"
	"github.com/golang-acexy/starter-redis/redismodule"
	"github.com/redis/go-redis/v9"
	"testing"
)

var moduleLoaders []declaration.ModuleLoader
var rModule *redismodule.RedisModule

func init() {
	rModule = &redismodule.RedisModule{
		RedisConfig: &redis.UniversalOptions{
			Addrs:    []string{":6379"},
			Password: "tech-acexy",
		},
	}
	moduleLoaders = []declaration.ModuleLoader{rModule}
}

func TestLoad(t *testing.T) {
	m := declaration.Module{ModuleLoaders: moduleLoaders}
	m.Load()
	select {}
}
