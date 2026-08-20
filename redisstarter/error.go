package redisstarter

import "errors"

var (
	ErrNilValue                 = errors.New("nil value")
	ErrNilKeys                  = errors.New("nil keys")
	ErrNotStringValue           = errors.New("not a string value")
	ErrAlreadySubscribedToTopic = errors.New("already subscribed to topic")
	ErrNotSubscribedToTopic     = errors.New("not subscribed to topic")
	ErrUnknownRedisResponse     = errors.New("unknown redis response")
	ErrRedisStarterAlreadyStarted = errors.New("redis starter already started")
	ErrRedisClientNotStarted      = errors.New("redis client not started")
	ErrRedisLockerNotStarted      = errors.New("redis locker not started")
	ErrNilLocker                  = errors.New("nil locker")
	ErrNilLockExecutable          = errors.New("nil lock executable")
	ErrInvalidLockTTL             = errors.New("invalid lock ttl")
	ErrInvalidLockRetryMax        = errors.New("invalid lock retry max")
	ErrInvalidLockRetryInterval   = errors.New("invalid lock retry interval")
)
