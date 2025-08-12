package redisstarter

import (
	"context"
	"github.com/acexy/golang-toolkit/logger"
	"github.com/bsm/redislock"
	"time"
)

type Locker struct {
	lock *redislock.Lock
}

func (l *Locker) Release() error {
	return l.lock.Release(context.Background())
}

func (l *Locker) RawLocker() *redislock.Lock {
	return l.lock
}

func (l *Locker) ReleaseWithCtx(ctx context.Context) error {
	return l.lock.Release(ctx)
}

func (l *Locker) Refresh(ttl time.Duration, opt *redislock.Options) error {
	return l.lock.Refresh(context.Background(), ttl, opt)
}

// 分布式锁

func distributedLocker() *redislock.Client {
	return redisLockerClient
}

func lock(ctx context.Context, key string, ttl time.Duration, executable func(), token string, retry redislock.RetryStrategy) (error, <-chan struct{}) {
	redisLock, err := distributedLocker().Obtain(ctx, key, ttl, &redislock.Options{
		RetryStrategy: retry,
		Token:         token,
	})
	if err != nil {
		return err, nil
	}
	chn := make(chan struct{})
	defer func() {
		err = redisLock.Release(context.Background())
		if err != nil {
			logger.Logrus().WithError(err).Errorln("release redisLock error key =", key)
		} else {
			close(chn)
		}
	}()
	executable()
	return nil, chn
}

// TryLock 尝试获取锁并执行executable函数
// request lockTtl: 获得锁之后的持续时长(超时自动释放)
func TryLock(key RedisKey, executable func(), opt *redislock.Options, keyAppend ...interface{}) (error, <-chan struct{}) {
	return TryLockWithContext(context.Background(), key, executable, opt, keyAppend...)
}

// TryAndGetLocker 尝试获取锁并返回Locker
func TryAndGetLocker(key RedisKey, opt *redislock.Options, keyAppend ...interface{}) (*Locker, error) {
	return TryAndGetLockerWithContext(context.Background(), key, opt, keyAppend...)
}

// TryLockWithContext 尝试获取锁并执行executable函数
func TryLockWithContext(ctx context.Context, key RedisKey, executable func(), opt *redislock.Options, keyAppend ...interface{}) (error, <-chan struct{}) {
	redisLock, err := distributedLocker().Obtain(ctx, key.RawKeyString(keyAppend...), key.Expire, opt)
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

// TryAndGetLockerWithContext 尝试获取锁并返回Locker
func TryAndGetLockerWithContext(ctx context.Context, key RedisKey, opt *redislock.Options, keyAppend ...interface{}) (*Locker, error) {
	redisLock, err := distributedLocker().Obtain(ctx, key.RawKeyString(keyAppend...), key.Expire, opt)
	if err != nil {
		return nil, err
	}
	return &Locker{
		lock: redisLock,
	}, nil
}

// LockWithMaxRetry 持续尝试获取锁
// request 	lockTtl 获得锁之后的持续时长(超时自动释放)
//
//	retryMax 尝试获取锁最大重试次数
//	intervalMil 重试间隔(millisecond)
func LockWithMaxRetry(ctx context.Context, key RedisKey, token string, retryMax, retryInterval int, executable func(), keyAppend ...interface{}) (error, <-chan struct{}) {
	retry := redislock.LimitRetry(redislock.LinearBackoff(time.Duration(retryInterval)*time.Millisecond), retryMax)
	return lock(ctx, key.RawKeyString(keyAppend...), key.Expire, executable, token, retry)
}

// LockWithDeadline 持续尝试获取锁
// request 	lockTtl 获得锁之后的持续时长(超时自动释放)
//
//	retryDeadline 重试持续时间
//	retryInterval 重试间隔(millisecond)
func LockWithDeadline(ctx context.Context, key RedisKey, token string, retryDeadline time.Time, retryInterval int, executable func(), keyAppend ...interface{}) (error, <-chan struct{}) {
	retry := redislock.LinearBackoff(time.Duration(retryInterval) * time.Millisecond)
	lockCtx, cancel := context.WithDeadline(ctx, retryDeadline)
	defer cancel()
	return lock(lockCtx, key.RawKeyString(keyAppend...), key.Expire, executable, token, retry)
}
