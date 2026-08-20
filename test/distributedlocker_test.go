package test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/acexy/golang-toolkit/math/random"
	"github.com/acexy/golang-toolkit/sys"
	"github.com/golang-acexy/starter-redis/redisstarter"
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
	err := redisstarter.TryLock(redisstarter.NewRedisKey("tryLock", time.Second), func() error {
		*i = *i + 1
		fmt.Println(*i)
		return nil
	})
	if err != nil {
		fmt.Printf("%+v %s \n", err, k)
		return
	}
}

func lock(ctx context.Context, key string, i *int) {
	err := redisstarter.LockWithDeadline(ctx, redisstarter.NewRedisKey("key", time.Minute), time.Now().Add(time.Minute), 200, func() error {
		*i = *i + 1
		time.Sleep(time.Duration(random.RandRangeInt(100, 300)) * time.Millisecond)
		fmt.Println(*i)
		return nil
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
	wg.Add(20)
	for i := 0; i < 20; i++ {
		go func() {
			defer wg.Done()
			err := redisstarter.LockWithDeadline(context.Background(), redisstarter.NewRedisKey("distributed-key", time.Minute), time.Now().Add(time.Minute*5), 200, func() error {
				executable()
				return nil
			})
			if err != nil {
				fmt.Printf("%+v %s \n", err, key)
				return
			}
			fmt.Println("done")
		}()
	}
	wg.Wait()
}

func TestTryAndGetLocker(t *testing.T) {
	tk := "distributed-key-locker" + time.Now().String()
	go func() {
		l, err := redisstarter.ObtainLocker(redisstarter.NewRedisKey(tk, time.Second*10))
		if err != nil {
			fmt.Println(err)
			return
		}
		time.Sleep(time.Second * 5)
		err = l.Release()
		if err != nil {
			fmt.Println(err)
			return
		}
	}()

	go func() {
		for {
			_, err := redisstarter.ObtainLocker(redisstarter.NewRedisKey(tk, time.Second))
			if err != nil {
				fmt.Println(err)
				time.Sleep(time.Millisecond * 200)
				continue
			}
			break
		}
	}()

	sys.ShutdownHolding()
}
