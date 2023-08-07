package redismodule

import (
	"context"
	"github.com/acexy/golang-toolkit/log"
	"github.com/bsm/redislock"
	"time"
)

func RawLockClient() *redislock.Client {
	return redisLockerClient
}

// 分布式锁

// TryLock 取锁并执行executable函数
// request time: 自身获取锁之后最大持有时间
func TryLock(key string, ttl time.Duration, executable func()) error {
	return TryLockWithContext(context.Background(), key, ttl, executable)
}

// TryLockWithContext 取锁并执行executable函数
func TryLockWithContext(ctx context.Context, key string, ttl time.Duration, executable func()) error {
	lock, err := redisLockerClient.Obtain(ctx, key, ttl, nil)
	if err != nil {
		return err
	}
	defer func() {
		err := lock.Release(context.Background())
		if err != nil {
			log.Logrus().WithError(err).Errorln("release lock error key =", key)
		}
	}()
	executable()
	return err
}

func lock(ctx context.Context, key string, ttl time.Duration, executable func(), retry redislock.RetryStrategy) error {
	lock, err := redisLockerClient.Obtain(ctx, key, ttl, &redislock.Options{
		RetryStrategy: retry,
	})
	if err != nil {
		return err
	}
	defer func() {
		err := lock.Release(context.Background())
		if err != nil {
			log.Logrus().WithError(err).Errorln("release lock error key =", key)
		}
	}()
	executable()
	return nil
}

// LockWithMaxRetry 持续尝试获取锁
// request 	maxRetry 最大重试次数
//			intervalMil 重试间隔(millisecond)
func LockWithMaxRetry(ctx context.Context, key string, ttl time.Duration, maxRetry, intervalMil int, executable func()) error {
	retry := redislock.LimitRetry(redislock.LinearBackoff(time.Duration(intervalMil)*time.Millisecond), maxRetry)
	return lock(ctx, key, ttl, executable, retry)
}

// LockWithDeadline 持续尝试获取锁
// request 	deadline 持续时长
//			intervalMil 重试间隔(millisecond)
func LockWithDeadline(ctx context.Context, key string, ttl time.Duration, deadline time.Time, intervalMil int, executable func()) error {
	retry := redislock.LinearBackoff(time.Duration(intervalMil) * time.Millisecond)
	lockCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	return lock(lockCtx, key, ttl, executable, retry)
}
