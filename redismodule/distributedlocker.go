package redismodule

import (
	"context"
	"github.com/acexy/golang-toolkit/logger"
	"github.com/bsm/redislock"
	"time"
)

// 分布式锁

func RawLockClient() *redislock.Client {
	return redisLockerClient
}

func lock(ctx context.Context, key string, ttl time.Duration, executable func(), retry redislock.RetryStrategy) error {
	redisLock, err := redisLockerClient.Obtain(ctx, key, ttl, &redislock.Options{
		RetryStrategy: retry,
	})
	if err != nil {
		return err
	}
	defer func() {
		err := redisLock.Release(context.Background())
		if err != nil {
			logger.Logrus().WithError(err).Errorln("release redisLock error key =", key)
		}
	}()
	executable()
	return nil
}

// TryLock 取锁并执行executable函数
// request lockTtl: 获得锁之后的持续时长(超时自动释放)
func TryLock(key string, lockTtl time.Duration, executable func()) error {
	return TryLockWithContext(context.Background(), key, lockTtl, executable)
}

// TryLockWithContext 取锁并执行executable函数
func TryLockWithContext(ctx context.Context, key string, lockTtl time.Duration, executable func()) error {
	redisLock, err := redisLockerClient.Obtain(ctx, key, lockTtl, nil)
	if err != nil {
		return err
	}
	defer func() {
		err := redisLock.Release(context.Background())
		if err != nil {
			logger.Logrus().WithError(err).Errorln("release redisLock error key =", key)
		}
	}()
	executable()
	return err
}

// LockWithMaxRetry 持续尝试获取锁
// request 	lockTtl 获得锁之后的持续时长(超时自动释放)
//
//	retryMax 尝试获取锁最大重试次数
//	intervalMil 重试间隔(millisecond)
func LockWithMaxRetry(ctx context.Context, key string, lockTtl time.Duration, retryMax, retryInterval int, executable func()) error {
	retry := redislock.LimitRetry(redislock.LinearBackoff(time.Duration(retryInterval)*time.Millisecond), retryMax)
	return lock(ctx, key, lockTtl, executable, retry)
}

// LockWithDeadline 持续尝试获取锁
// request 	lockTtl 获得锁之后的持续时长(超时自动释放)
//
//	retryDeadline 重试持续时间
//	retryInterval 重试间隔(millisecond)
func LockWithDeadline(ctx context.Context, key string, lockTtl time.Duration, retryDeadline time.Time, retryInterval int, executable func()) error {
	retry := redislock.LinearBackoff(time.Duration(retryInterval) * time.Millisecond)
	lockCtx, cancel := context.WithDeadline(ctx, retryDeadline)
	defer cancel()
	return lock(lockCtx, key, lockTtl, executable, retry)
}
