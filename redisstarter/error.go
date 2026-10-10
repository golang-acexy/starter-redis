package redisstarter

import "errors"

var (
	// ErrNilValue reports empty batch data or a nil interface value where explicitly checked.
	ErrNilValue = errors.New("nil value")
	// ErrNilKeys reports an empty string batch-read key list.
	ErrNilKeys = errors.New("nil keys")
	// ErrNotStringValue reports an unexpected non-string batch-read value.
	ErrNotStringValue = errors.New("not a string value")
	// ErrAlreadySubscribedToTopic reports an existing or pending topic rejected by basic Subscribe.
	ErrAlreadySubscribedToTopic = errors.New("already subscribed to topic")
	// ErrNotSubscribedToTopic reports Unsubscribe for an absent topic.
	ErrNotSubscribedToTopic = errors.New("not subscribed to topic")
	// ErrSubscriptionClosed reports subscription startup invalidated by topic cleanup.
	ErrSubscriptionClosed = errors.New("subscription closed during startup")
	// ErrUnknownRedisResponse reports an unsupported Bloom info reply shape or field value.
	ErrUnknownRedisResponse = errors.New("unknown redis response")
	// ErrRedisStarterAlreadyStarted reports another active Redis lifecycle.
	ErrRedisStarterAlreadyStarted = errors.New("redis starter already started")
	// ErrRedisClientNotStarted reports runtime absence at checked subscription or stop entry points.
	ErrRedisClientNotStarted = errors.New("redis client not started")
	// ErrRedisLockerNotStarted reports runtime absence during lock acquisition.
	ErrRedisLockerNotStarted = errors.New("redis locker not started")
	// ErrNilLocker reports operations on a nil or empty Locker.
	ErrNilLocker = errors.New("nil locker")
	// ErrNilLockExecutable reports a missing lock callback.
	ErrNilLockExecutable = errors.New("nil lock executable")
	// ErrInvalidLockTTL reports nonpositive lease duration during acquisition.
	ErrInvalidLockTTL = errors.New("invalid lock ttl")
	// ErrInvalidLockRetryMax reports a negative bounded retry count.
	ErrInvalidLockRetryMax = errors.New("invalid lock retry max")
	// ErrInvalidLockRetryInterval reports a nonpositive retry interval in milliseconds.
	ErrInvalidLockRetryInterval = errors.New("invalid lock retry interval")
)
