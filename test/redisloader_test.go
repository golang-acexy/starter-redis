package test

import (
	"context"
	"fmt"
	"github.com/acexy/golang-toolkit/math/random"
	"github.com/golang-acexy/starter-parent/parentmodule/declaration"
	"github.com/golang-acexy/starter-redis/redismodule"
	"github.com/redis/go-redis/v9"
	"testing"
	"time"
)

var moduleLoaders []declaration.ModuleLoader
var rModule *redismodule.RedisModule

// 单实例Redis
func TestStandalone(t *testing.T) {

	rModule = &redismodule.RedisModule{
		RedisConfig: &redis.UniversalOptions{
			Addrs:    []string{":6379"},
			Password: "tech-acexy",
		},
		RedisInterceptor: func(instance redis.UniversalClient) {
			fmt.Println(instance.PoolStats())
		},
	}
	moduleLoaders = []declaration.ModuleLoader{rModule}

	m = declaration.Module{ModuleLoaders: moduleLoaders}

	err := m.Load()
	if err != nil {
		fmt.Printf("%+v\n", err)
	}

	// 启动一批协程，并执行延迟sql，模拟并发多连接执行中场景
	go func() {
		for i := 1; i <= 10; i++ {
			go func() {
				for {
					err = redismodule.Set(context.Background(), redismodule.RedisKey(random.RandString(5)), random.RandString(5))
					if err != nil {
						fmt.Printf("%+v", err)
					}
				}
			}()
		}
	}()

	time.Sleep(time.Second * 3)
	fmt.Println(rModule.Unregister(10))
}

// 集群Redis
func TestCluster(t *testing.T) {

	rModule = &redismodule.RedisModule{
		RedisConfig: &redis.UniversalOptions{
			Addrs:    []string{":6379", ":6381", ":6380"},
			Password: "tech-acexy",
		},
		RedisInterceptor: func(instance redis.UniversalClient) {
			fmt.Println(instance.PoolStats())
		},
	}
	moduleLoaders = []declaration.ModuleLoader{rModule}

	m = declaration.Module{ModuleLoaders: moduleLoaders}

	err := m.Load()
	if err != nil {
		fmt.Printf("%+v\n", err)
	}

	// 启动一批协程，并执行延迟sql，模拟并发多连接执行中场景
	go func() {
		for i := 1; i <= 10; i++ {
			go func() {
				for {
					err = redismodule.Set(context.Background(), redismodule.RedisKey(random.RandString(5)), random.RandString(5))
					if err != nil {
						fmt.Printf("%+v", err)
					}
				}
			}()
		}
	}()

	time.Sleep(time.Second * 3)
	fmt.Println(rModule.Unregister(10))
}
