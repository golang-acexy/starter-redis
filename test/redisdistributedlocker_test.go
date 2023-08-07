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

func TestTryLock(t *testing.T) {
	number := 0
	key := random.RandString(5)
	for i := 0; i < 100; i++ {
		//go tryLock(key+strconv.Itoa(i), &number)
		go tryLock(key, &number)
	}
	time.Sleep(time.Second * 10)
	fmt.Println(number)
}

func tryLock(k string, i *int) {
	err := redismodule.TryLock(k, time.Minute, func() {
		*i = *i + 1
	})
	if err != nil {
		fmt.Printf("%+v %s \n", err, k)
		return
	}
	fmt.Println(*i)
}

func lock(ctx context.Context, key string, i *int) {
	err := redismodule.LockWithDeadline(ctx, key, time.Minute, time.Now().Add(time.Minute), 200, func() {
		*i = *i + 1
	})
	if err != nil {
		fmt.Printf("%+v %s \n", err, key)
		return
	}
	fmt.Println(*i)
}

func TestLockWithDeadline(t *testing.T) {
	ctx := context.Background()
	number := 0
	key := random.RandString(5)
	for i := 0; i < 100; i++ {
		go lock(ctx, key, &number)
	}
	time.Sleep(time.Second * 5)
	fmt.Println(number)
}

func TestDistributed(t *testing.T) {
	rModule = &redismodule.RedisModule{
		RedisConfig: &redis.UniversalOptions{
			Addrs:    []string{":6379"},
			Password: "tech-acexy",
		},
	}
	moduleLoaders = []declaration.ModuleLoader{rModule}

	m := declaration.Module{ModuleLoaders: moduleLoaders}

	err := m.Load()
	if err != nil {
		fmt.Printf("%+v\n", err)
	}
}
