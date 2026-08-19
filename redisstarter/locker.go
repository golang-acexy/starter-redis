package redisstarter

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/bsm/redislock"
)

const defaultLockerOperationTimeout = 5 * time.Second

var redisLockerClientMutex sync.RWMutex

// Locker 封装一个由当前调用者持有的分布式锁。
type Locker struct {
	lock *redislock.Lock
}

// Release 使用有限超时释放锁。
func (l *Locker) Release() error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultLockerOperationTimeout)
	defer cancel()
	return l.ReleaseWithContext(ctx)
}

// ReleaseWithContext 使用指定上下文释放锁。
func (l *Locker) ReleaseWithContext(ctx context.Context) error {
	if l == nil || l.lock == nil {
		return ErrNilLocker
	}
	return l.lock.Release(ctx)
}

// Refresh 使用有限超时刷新锁的 TTL。
func (l *Locker) Refresh(ttl time.Duration, opt *redislock.Options) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultLockerOperationTimeout)
	defer cancel()
	return l.RefreshWithContext(ctx, ttl, opt)
}

// RefreshWithContext 使用指定上下文刷新锁的 TTL。
func (l *Locker) RefreshWithContext(ctx context.Context, ttl time.Duration, opt *redislock.Options) error {
	if l == nil || l.lock == nil {
		return ErrNilLocker
	}
	return l.lock.Refresh(ctx, ttl, opt)
}

// TTL 使用有限超时查询锁的剩余有效期。
func (l *Locker) TTL() (time.Duration, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultLockerOperationTimeout)
	defer cancel()
	return l.TTLWithContext(ctx)
}

// TTLWithContext 使用指定上下文查询锁的剩余有效期。
func (l *Locker) TTLWithContext(ctx context.Context) (time.Duration, error) {
	if l == nil || l.lock == nil {
		return 0, ErrNilLocker
	}
	return l.lock.TTL(ctx)
}

func currentLockerClient() (*redislock.Client, error) {
	client := rawLockerClient()
	if client == nil {
		return nil, ErrRedisLockerNotStarted
	}
	return client, nil
}

func rawLockerClient() *redislock.Client {
	redisLockerClientMutex.RLock()
	defer redisLockerClientMutex.RUnlock()
	return redisLockerClient
}

func setLockerClient(client *redislock.Client) {
	redisLockerClientMutex.Lock()
	defer redisLockerClientMutex.Unlock()
	redisLockerClient = client
}

func executeWithLock(ctx context.Context, key string, ttl time.Duration, opt *redislock.Options, executable func() error) (err error) {
	if executable == nil {
		return ErrNilLockExecutable
	}
	if ttl <= 0 {
		return ErrInvalidLockTTL
	}
	lockerClient, err := currentLockerClient()
	if err != nil {
		return err
	}
	redisLock, err := lockerClient.Obtain(ctx, key, ttl, opt)
	if err != nil {
		return err
	}
	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), defaultLockerOperationTimeout)
		defer cancel()
		if releaseErr := redisLock.Release(releaseCtx); releaseErr != nil {
			err = errors.Join(err, releaseErr)
		}
	}()
	return executable()
}

// TryLock 尝试获取锁并同步执行 executable，返回执行或释放锁产生的错误。
func TryLock(key RedisKey, executable func() error, keyAppend ...any) error {
	return TryLockWithContext(context.Background(), key, executable, keyAppend...)
}

// ObtainLocker 尝试获取由调用者管理生命周期的锁。
func ObtainLocker(key RedisKey, keyAppend ...any) (*Locker, error) {
	return ObtainLockerWithContext(context.Background(), key, keyAppend...)
}

// TryLockWithOptions 使用高级选项尝试获取锁并同步执行 executable。
func TryLockWithOptions(key RedisKey, opt *redislock.Options, executable func() error, keyAppend ...any) error {
	return TryLockWithContextOptions(context.Background(), key, opt, executable, keyAppend...)
}

// ObtainLockerWithOptions 使用高级选项获取由调用者管理生命周期的锁。
func ObtainLockerWithOptions(key RedisKey, opt *redislock.Options, keyAppend ...any) (*Locker, error) {
	return ObtainLockerWithContextOptions(context.Background(), key, opt, keyAppend...)
}

// TryLockWithContext 使用指定上下文尝试获取锁并同步执行 executable。
// ctx 仅控制获取锁；锁的有效期由 key.Expire 决定。
func TryLockWithContext(ctx context.Context, key RedisKey, executable func() error, keyAppend ...any) error {
	return TryLockWithContextOptions(ctx, key, nil, executable, keyAppend...)
}

// TryLockWithContextOptions 使用指定上下文和高级选项获取锁并同步执行 executable。
func TryLockWithContextOptions(ctx context.Context, key RedisKey, opt *redislock.Options, executable func() error, keyAppend ...any) error {
	return executeWithLock(ctx, key.RawKeyString(keyAppend...), key.Expire, opt, executable)
}

// ObtainLockerWithContext 使用指定上下文获取由调用者管理生命周期的锁。
func ObtainLockerWithContext(ctx context.Context, key RedisKey, keyAppend ...any) (*Locker, error) {
	return ObtainLockerWithContextOptions(ctx, key, nil, keyAppend...)
}

// ObtainLockerWithContextOptions 使用指定上下文和高级选项获取由调用者管理生命周期的锁。
func ObtainLockerWithContextOptions(ctx context.Context, key RedisKey, opt *redislock.Options, keyAppend ...any) (*Locker, error) {
	if key.Expire <= 0 {
		return nil, ErrInvalidLockTTL
	}
	lockerClient, err := currentLockerClient()
	if err != nil {
		return nil, err
	}
	redisLock, err := lockerClient.Obtain(ctx, key.RawKeyString(keyAppend...), key.Expire, opt)
	if err != nil {
		return nil, err
	}
	return &Locker{
		lock: redisLock,
	}, nil
}

// LockWithMaxRetry 按固定间隔重试获取锁，并同步执行 executable。
func LockWithMaxRetry(ctx context.Context, key RedisKey, retryMax, retryInterval int, executable func() error, keyAppend ...any) error {
	return LockWithMaxRetryOptions(ctx, key, nil, retryMax, retryInterval, executable, keyAppend...)
}

// LockWithMaxRetryOptions 使用高级选项按固定间隔重试获取锁，并同步执行 executable。
func LockWithMaxRetryOptions(ctx context.Context, key RedisKey, opt *redislock.Options, retryMax, retryInterval int, executable func() error, keyAppend ...any) error {
	if retryMax < 0 {
		return ErrInvalidLockRetryMax
	}
	if retryInterval <= 0 {
		return ErrInvalidLockRetryInterval
	}
	retry := redislock.LimitRetry(redislock.LinearBackoff(time.Duration(retryInterval)*time.Millisecond), retryMax)
	return executeWithLock(ctx, key.RawKeyString(keyAppend...), key.Expire, optionsWithRetry(opt, retry), executable)
}

// LockWithDeadline 在截止时间前按固定间隔重试获取锁，并同步执行 executable。
func LockWithDeadline(ctx context.Context, key RedisKey, retryDeadline time.Time, retryInterval int, executable func() error, keyAppend ...any) error {
	return LockWithDeadlineOptions(ctx, key, nil, retryDeadline, retryInterval, executable, keyAppend...)
}

// LockWithDeadlineOptions 使用高级选项在截止时间前重试获取锁，并同步执行 executable。
func LockWithDeadlineOptions(ctx context.Context, key RedisKey, opt *redislock.Options, retryDeadline time.Time, retryInterval int, executable func() error, keyAppend ...any) error {
	if retryInterval <= 0 {
		return ErrInvalidLockRetryInterval
	}
	retry := redislock.LinearBackoff(time.Duration(retryInterval) * time.Millisecond)
	lockCtx, cancel := context.WithDeadline(ctx, retryDeadline)
	defer cancel()
	return executeWithLock(lockCtx, key.RawKeyString(keyAppend...), key.Expire, optionsWithRetry(opt, retry), executable)
}

func optionsWithRetry(opt *redislock.Options, retry redislock.RetryStrategy) *redislock.Options {
	if opt == nil {
		return &redislock.Options{RetryStrategy: retry}
	}
	result := *opt
	result.RetryStrategy = retry
	return &result
}
