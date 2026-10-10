package redisstarter

import (
	"context"
	"errors"
	"time"

	"github.com/bsm/redislock"
)

const defaultLockerOperationTimeout = 5 * time.Second

// Locker represents a caller-owned lease backed by the starter-owned client.
// It does not auto-renew or fence external writes; expiry can permit another owner
// while business code still runs. Coordinate release and refresh within one owner,
// and keep driver options and retry strategies safe from concurrent mutation.
type Locker struct {
	lock *redislock.Lock
}

// Release releases the owned lease using a five-second context; a nil or empty wrapper returns
// ErrNilLocker. Repeated release can return redislock.ErrLockNotHeld.
func (l *Locker) Release() error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultLockerOperationTimeout)
	defer cancel()
	return l.ReleaseWithContext(ctx)
}

// ReleaseWithContext releases the owned lease with a non-nil context; a nil wrapper returns ErrNilLocker
// and an expired or already-released lease can return redislock.ErrLockNotHeld.
func (l *Locker) ReleaseWithContext(ctx context.Context) error {
	if l == nil || l.lock == nil {
		return ErrNilLocker
	}
	return l.lock.Release(ctx)
}

// Refresh refreshes the owned lease using a five-second context and driver options. Supply a positive
// duration with millisecond precision; this wrapper does not validate it. A nil wrapper returns
// ErrNilLocker.
func (l *Locker) Refresh(ttl time.Duration, opt *redislock.Options) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultLockerOperationTimeout)
	defer cancel()
	return l.RefreshWithContext(ctx, ttl, opt)
}

// RefreshWithContext refreshes the owned lease using a non-nil context and driver options. Duration is
// forwarded without validation and truncated to milliseconds; a lost lease cannot be recovered by
// refreshing.
func (l *Locker) RefreshWithContext(ctx context.Context, ttl time.Duration, opt *redislock.Options) error {
	if l == nil || l.lock == nil {
		return ErrNilLocker
	}
	return l.lock.Refresh(ctx, ttl, opt)
}

// TTL reads remaining lease time using a five-second context; zero indicates it is no longer owned. A
// nil wrapper returns ErrNilLocker.
func (l *Locker) TTL() (time.Duration, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultLockerOperationTimeout)
	defer cancel()
	return l.TTLWithContext(ctx)
}

// TTLWithContext reads remaining owned-lease time using a non-nil context; zero indicates expiry or loss
// of ownership. A nil wrapper returns ErrNilLocker.
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
	runtime := redisRuntimeState.Load()
	if runtime == nil {
		return nil
	}
	return runtime.locker
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
	// Acquisition cancellation must not prevent the release attempt.
	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), defaultLockerOperationTimeout)
		defer cancel()
		if releaseErr := redisLock.Release(releaseCtx); releaseErr != nil {
			err = errors.Join(err, releaseErr)
		}
	}()
	return executable()
}

// TryLock obtains a positive key.Expire lease with default no-retry options and runs a non-nil callback
// synchronously. It always attempts release, joining callback and release errors; callback panics
// propagate after cleanup. No automatic renewal occurs. If the context has no deadline, the driver caps
// acquisition waiting by the lease duration.
func TryLock(key RedisKey, executable func() error, keyAppend ...any) error {
	return TryLockWithContext(context.Background(), key, executable, keyAppend...)
}

// ObtainLocker obtains a positive key.Expire lease with default no-retry options and a background
// context. The caller must release or refresh the returned lease; no automatic renewal occurs. If the
// context has no deadline, the driver caps acquisition waiting by the lease duration.
func ObtainLocker(key RedisKey, keyAppend ...any) (*Locker, error) {
	return ObtainLockerWithContext(context.Background(), key, keyAppend...)
}

// TryLockWithOptions obtains a positive lease using driver options and a background context, then
// executes and releases as in TryLock. Retry options can extend waiting; keep options stable and do not
// share mutable retry state between acquisitions. If the context has no deadline, the driver caps
// acquisition waiting by the lease duration.
func TryLockWithOptions(key RedisKey, opt *redislock.Options, executable func() error, keyAppend ...any) error {
	return TryLockWithContextOptions(context.Background(), key, opt, executable, keyAppend...)
}

// ObtainLockerWithOptions obtains a caller-owned positive lease with driver options and a background
// context. Retry options control waiting; the caller must release or refresh it. If the context has no
// deadline, the driver caps acquisition waiting by the lease duration.
func ObtainLockerWithOptions(key RedisKey, opt *redislock.Options, keyAppend ...any) (*Locker, error) {
	return ObtainLockerWithContextOptions(context.Background(), key, opt, keyAppend...)
}

// TryLockWithContext runs and releases a non-nil callback after acquiring a positive lease. The non-nil
// context controls acquisition only, not callback execution or automatic renewal; release uses a
// separate five-second context. If the context has no deadline, the driver caps acquisition waiting by
// the lease duration.
func TryLockWithContext(ctx context.Context, key RedisKey, executable func() error, keyAppend ...any) error {
	return TryLockWithContextOptions(ctx, key, nil, executable, keyAppend...)
}

// TryLockWithContextOptions acquires with a non-nil context and driver options, then runs a non-nil
// callback. Positive key.Expire controls the lease independently of callback duration. Release uses a
// fresh five-second context and joins errors; panics are not recovered. If the context has no deadline,
// the driver caps acquisition waiting by the lease duration.
func TryLockWithContextOptions(ctx context.Context, key RedisKey, opt *redislock.Options, executable func() error, keyAppend ...any) error {
	return executeWithLock(ctx, key.RawKeyString(keyAppend...), key.Expire, opt, executable)
}

// ObtainLockerWithContext obtains a caller-owned positive lease using a non-nil acquisition context.
// Cancellation does not automatically release an acquired lease; use Release explicitly. If the context
// has no deadline, the driver caps acquisition waiting by the lease duration.
func ObtainLockerWithContext(ctx context.Context, key RedisKey, keyAppend ...any) (*Locker, error) {
	return ObtainLockerWithContextOptions(ctx, key, nil, keyAppend...)
}

// ObtainLockerWithContextOptions obtains a caller-owned positive lease using a non-nil context and
// stable driver options. Missing runtime returns ErrRedisLockerNotStarted; the caller must release or
// refresh the lease. If the context has no deadline, the driver caps acquisition waiting by the lease
// duration.
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

// LockWithMaxRetry acquires with at most retryMax retries at retryInterval milliseconds, then executes
// and releases as in TryLock. The context must be non-nil; retryMax must be nonnegative and
// retryInterval positive. Zero retries means one initial attempt. If the context has no deadline, the
// driver caps acquisition waiting by the lease duration.
func LockWithMaxRetry(ctx context.Context, key RedisKey, retryMax, retryInterval int, executable func() error, keyAppend ...any) error {
	return LockWithMaxRetryOptions(ctx, key, nil, retryMax, retryInterval, executable, keyAppend...)
}

// LockWithMaxRetryOptions copies opt and replaces its retry policy with bounded linear backoff.
// Nonnegative retryMax and positive millisecond retryInterval are required; the non-nil context bounds
// acquisition only, not callback execution. If the context has no deadline, the driver caps acquisition
// waiting by the lease duration.
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

// LockWithDeadline retries acquisition until retryDeadline using a positive millisecond retryInterval
// and a non-nil context. The deadline bounds acquisition, not callback execution or lease renewal.
func LockWithDeadline(ctx context.Context, key RedisKey, retryDeadline time.Time, retryInterval int, executable func() error, keyAppend ...any) error {
	return LockWithDeadlineOptions(ctx, key, nil, retryDeadline, retryInterval, executable, keyAppend...)
}

// LockWithDeadlineOptions copies opt and replaces its retry policy with linear backoff under
// retryDeadline. A past deadline can prevent acquisition; callback execution is not forcibly canceled
// and release uses a separate context.
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
