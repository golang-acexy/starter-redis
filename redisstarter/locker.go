package redisstarter

import (
	"context"
	"github.com/acexy/golang-toolkit/logger"
	"github.com/bsm/redislock"
	"time"
)

// 分布式锁

func distributedLocker() *redislock.Client {
	return redisLockerClient
}

func lock(ctx context.Context, key string, ttl time.Duration, executable func(), retry redislock.RetryStrategy) (error, <-chan struct{}) {
	redisLock, err := distributedLocker().Obtain(ctx, key, ttl, &redislock.Options{
		RetryStrategy: retry,
	})
	if err != nil {
		return err, nil
	}
	chn := make(chan struct{})
	defer func() {
		close(chn)
		err = redisLock.Release(context.Background())
		if err != nil {
			logger.Logrus().WithError(err).Errorln("release redisLock error key =", key)
		}
	}()
	executable()
	return nil, chn
}

// TryLock 尝试获取锁并执行executable函数
// request lockTtl: 获得锁之后的持续时长(超时自动释放)
func TryLock(key RedisKey, executable func(), keyAppend ...interface{}) (error, <-chan struct{}) {
	return TryLockWithContext(context.Background(), key, executable, keyAppend...)
}

// TryLockWithContext 尝试获取锁并执行executable函数
func TryLockWithContext(ctx context.Context, key RedisKey, executable func(), keyAppend ...interface{}) (error, <-chan struct{}) {
	redisLock, err := distributedLocker().Obtain(ctx, OriginKeyString(key.KeyFormat, keyAppend...), key.Expire, nil)
	if err != nil {
		return err, nil
	}
	chn := make(chan struct{})
	defer func() {
		close(chn)
		err = redisLock.Release(context.Background())
		if err != nil {
			logger.Logrus().WithError(err).Errorln("release redisLock error key =", key)
		}
	}()
	executable()
	return err, chn
}

// LockWithMaxRetry 持续尝试获取锁
// request 	lockTtl 获得锁之后的持续时长(超时自动释放)
//
//	retryMax 尝试获取锁最大重试次数
//	intervalMil 重试间隔(millisecond)
func LockWithMaxRetry(ctx context.Context, key RedisKey, retryMax, retryInterval int, executable func(), keyAppend ...interface{}) (error, <-chan struct{}) {
	retry := redislock.LimitRetry(redislock.LinearBackoff(time.Duration(retryInterval)*time.Millisecond), retryMax)
	return lock(ctx, OriginKeyString(key.KeyFormat, keyAppend...), key.Expire, executable, retry)
}

// LockWithDeadline 持续尝试获取锁
// request 	lockTtl 获得锁之后的持续时长(超时自动释放)
//
//	retryDeadline 重试持续时间
//	retryInterval 重试间隔(millisecond)
func LockWithDeadline(ctx context.Context, key RedisKey, retryDeadline time.Time, retryInterval int, executable func(), keyAppend ...interface{}) (error, <-chan struct{}) {
	retry := redislock.LinearBackoff(time.Duration(retryInterval) * time.Millisecond)
	lockCtx, cancel := context.WithDeadline(ctx, retryDeadline)
	defer cancel()
	return lock(lockCtx, OriginKeyString(key.KeyFormat, keyAppend...), key.Expire, executable, retry)
}
