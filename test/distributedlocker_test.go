package test

import (
	"context"
	"fmt"
	"github.com/acexy/golang-toolkit/math/random"
	"github.com/golang-acexy/starter-redis/redismodule"
	"testing"
	"time"
)

func TestTryLock(t *testing.T) {
	number := 0
	key := random.RandString(5)
	for i := 0; i < 100; i++ {
		go tryLock(key, &number)
	}
	time.Sleep(time.Second * 5)
	fmt.Println(number)
}

func tryLock(k string, i *int) {
	err := redismodule.TryLock(k, time.Minute, func() {
		*i = *i + 1
		fmt.Println(*i)
	})
	if err != nil {
		fmt.Printf("%+v %s \n", err, k)
		return
	}
}

func lock(ctx context.Context, key string, i *int) {
	err := redismodule.LockWithDeadline(ctx, key, time.Minute, time.Now().Add(time.Minute), 200, func() {
		*i = *i + 1
		time.Sleep(time.Duration(random.RandRangeInt(100, 300)) * time.Millisecond)
		fmt.Println(*i)
	})
	if err != nil {
		fmt.Printf("%+v %s \n", err, key)
		return
	}
}

func TestLockWithDeadline(t *testing.T) {
	ctx := context.Background()
	deadline, cancel := context.WithDeadline(ctx, time.Now().Add(10*time.Second))
	number := 0
	key := random.RandString(5)
	for i := 0; i < 100; i++ {
		go lock(deadline, key, &number)
	}
	time.Sleep(time.Second * 5)
	cancel()
	fmt.Println(number)
}

func TestMuxLockClient(t *testing.T) {
	TestTryLock(t)
	TestLockWithDeadline(t)
}

func executable() {
	time.Sleep(time.Duration(random.RandRangeInt(100, 300)) * time.Millisecond)
	ctx := context.Background()

	key1 := redismodule.RedisKey{
		KeyFormat: "redis-key",
	}

	var v int
	err := redismodule.StringCmd().GetAny(ctx, key1, &v)
	if err != nil {
		fmt.Println(err)
	}
	v += 1
	fmt.Println("set ", v, "into redis")
	err = redismodule.StringCmd().SetAny(ctx, key1, v)
	if err != nil {
		fmt.Println(err)
	}
}

func TestExecutable(t *testing.T) {
	executable()
}

// 快速执行多次该方法，模拟多实例分布式锁
func TestDistributedLock(t *testing.T) {
	key := "distributed-key"
	for i := 0; i < 100; i++ {
		go func() {
			err := redismodule.LockWithDeadline(context.Background(), key, time.Minute, time.Now().Add(time.Minute*5), 200, executable)
			if err != nil {
				fmt.Printf("%+v %s \n", err, key)
				return
			}
		}()
	}
	time.Sleep(time.Second * 40)
}
