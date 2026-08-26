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

func ListCmd() *cmdList {
	return listCmd
}

// LLen 获取队列长度
func (*cmdList) LLen(key RedisKey, keyAppend ...any) int64 {
	result := redisClient.LLen(context.Background(), key.RawKeyString(keyAppend...))
	if result.Err() != nil {
		return 0
	}
	return result.Val()
}

// Push 数据入队
func (*cmdList) Push(directionRight bool, key RedisKey, data string, keyAppend ...any) error {
	if directionRight {
		return redisClient.RPush(context.Background(), key.RawKeyString(keyAppend...), data).Err()
	}
	return redisClient.LPush(context.Background(), key.RawKeyString(keyAppend...), data).Err()
}

// BPop 数据出队
// directionRight: true 从右出，false 从左出
// timeout: 向队列获取数据的最大等待时间，0 为永久阻塞
func (*cmdList) BPop(ctx context.Context, directionRight bool, timeout time.Duration, key RedisKey, keyAppend ...any) <-chan string {
	keyString := key.RawKeyString(keyAppend...)
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
					data, err = redisClient.BRPop(ctx, timeout, keyString).Result()
				} else {
					data, err = redisClient.BLPop(ctx, timeout, keyString).Result()
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
