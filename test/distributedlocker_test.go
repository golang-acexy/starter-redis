package test

import (
	"context"
	"fmt"
	"github.com/acexy/golang-toolkit/math/random"
	"github.com/golang-acexy/starter-redis/redisstarter"
	"sync"
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
	err, done := redisstarter.TryLock(redisstarter.NewRedisKey("tryLock", time.Second), func() {
		*i = *i + 1
		fmt.Println(*i)
	})
	if err != nil {
		fmt.Printf("%+v %s \n", err, k)
		return
	}
	<-done
}

func lock(ctx context.Context, key string, i *int) {
	err, done := redisstarter.LockWithDeadline(ctx, redisstarter.NewRedisKey("key", time.Minute), time.Now().Add(time.Minute), 200, func() {
		*i = *i + 1
		time.Sleep(time.Duration(random.RandRangeInt(100, 300)) * time.Millisecond)
		fmt.Println(*i)
	})
	if err != nil {
		fmt.Printf("%+v %s \n", err, key)
		return
	}
	<-done
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
	key1 := redisstarter.RedisKey{
		KeyFormat: "redis-key",
	}
	var v int
	err := redisstarter.StringCmd().GetAny(key1, &v)
	if err != nil {
		fmt.Println(err)
	}
	v += 1
	fmt.Println("set ", v, "into redis")
	err = redisstarter.StringCmd().SetAny(key1, v)
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
	var wg sync.WaitGroup
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			err, done := redisstarter.LockWithDeadline(context.Background(), redisstarter.NewRedisKey("distributed-key", time.Minute), time.Now().Add(time.Minute*5), 200, executable)
			if err != nil {
				fmt.Printf("%+v %s \n", err, key)
				return
			}
			<-done
			fmt.Println("done")
		}()
	}
	wg.Wait()
}
