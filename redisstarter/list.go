package redisstarter

import (
	"context"
	"errors"
	"time"

	"github.com/acexy/golang-toolkit/logger"
	"github.com/redis/go-redis/v9"
)

type cmdList struct {
}

var listCmd = new(cmdList)

// ListCmd returns shared list helpers. Calls require a running starter and can panic otherwise. Push and
// length use background contexts; none of these methods apply key.Expire.
func ListCmd() *cmdList {
	return listCmd
}

// LLen returns list length, zero for an absent key, and also zero on driver error.
func (*cmdList) LLen(key RedisKey, keyAppend ...any) int64 {
	result := rawRedisClient().LLen(context.Background(), key.RawKeyString(keyAppend...))
	if result.Err() != nil {
		return 0
	}
	return result.Val()
}

// Push pushes one value to the right when directionRight is true, otherwise left. It does not apply
// key.Expire.
func (*cmdList) Push(directionRight bool, key RedisKey, data string, keyAppend ...any) error {
	if directionRight {
		return rawRedisClient().RPush(context.Background(), key.RawKeyString(keyAppend...), data).Err()
	}
	return rawRedisClient().LPush(context.Background(), key.RawKeyString(keyAppend...), data).Err()
}

// BPop starts an unbuffered pop stream using a non-nil context. True pops right, false pops left; zero
// timeout asks Redis to block indefinitely. Errors are logged and retried, with a five-second delay
// after non-timeout failures. Drain the stream or cancel; in-flight read interruption depends on driver
// settings. A popped item can be lost if cancellation prevents channel delivery.
func (*cmdList) BPop(ctx context.Context, directionRight bool, timeout time.Duration, key RedisKey, keyAppend ...any) <-chan string {
	keyString := key.RawKeyString(keyAppend...)
	client := rawRedisClient()
	c := make(chan string)
	go func() {
		defer close(c)
		exception := false
		for {
			if exception {
				logger.Logrus().Warningln("BPop caught an exception, now sleeping for 5 seconds before retrying")
				timer := time.NewTimer(5 * time.Second)
				select {
				case <-ctx.Done():
					timer.Stop()
					return
				case <-timer.C:
				}
			}
			select {
			case <-ctx.Done():
				return
			default:
				var data []string
				var err error
				if directionRight {
					data, err = client.BRPop(ctx, timeout, keyString).Result()
				} else {
					data, err = client.BLPop(ctx, timeout, keyString).Result()
				}
				if err == nil && len(data) > 1 {
					select {
					case c <- data[1]:
						exception = false
					case <-ctx.Done():
						return
					}
				} else if err == nil {
					exception = true
					logger.Logrus().Errorln("BPop received an invalid Redis response")
				} else {
					if !errors.Is(err, redis.Nil) && !errors.Is(err, context.Canceled) {
						exception = true
						logger.Logrus().WithError(err).Errorln("BPop catch error", err)
					}
				}
			}
		}
	}()
	return c
}
