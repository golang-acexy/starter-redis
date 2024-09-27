package redisstarter

import (
	"context"
	"errors"
	"github.com/acexy/golang-toolkit/logger"
	"github.com/redis/go-redis/v9"
	"time"
)

type cmdList struct {
}

var listCmd = new(cmdList)

func ListCmd() *cmdList {
	return listCmd
}

// LLen 获取队列长度
func (*cmdList) LLen(key RedisKey, keyAppend ...interface{}) int64 {
	result := redisClient.LLen(context.Background(), OriginKeyString(key.KeyFormat, keyAppend...))
	if result.Err() != nil {
		return 0
	}
	return result.Val()
}

// Push 数据入队
func (*cmdList) Push(directionRight bool, key RedisKey, data string, keyAppend ...interface{}) error {
	if directionRight {
		return redisClient.RPush(context.Background(), OriginKeyString(key.KeyFormat, keyAppend...), data).Err()
	}
	return redisClient.LPush(context.Background(), OriginKeyString(key.KeyFormat, keyAppend...), data).Err()
}

// BPop 数据出队
// directionRight: true 从右出，false 从左出
// timeout: 向队列获取数据的最大等待时间，0 为永久阻塞
func (*cmdList) BPop(ctx context.Context, directionRight bool, timeout time.Duration, key RedisKey, keyAppend ...interface{}) <-chan string {
	keyString := OriginKeyString(key.KeyFormat, keyAppend...)
	c := make(chan string)
	go func() {
		defer close(c)
		exception := false
		for {
			if exception {
				logger.Logrus().Warningln("BPop caught an exception, now sleeping for 5 seconds before retrying")
				time.Sleep(time.Second * 5)
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
				if err == nil {
					c <- data[1]
					exception = false
				} else {
					if !errors.Is(err, redis.Nil) && !errors.Is(err, context.Canceled) {
						exception = true
						logger.Logrus().WithError(err).Errorln("BPop catch error", err)
					}
				}
			}
		}
	}()
	if timeout == time.Duration(0) {
		// 该逻辑是为了防止使用永久阻塞式弹出数据的方式将导致上面的select无法感知上下文取消
		// 通过补偿来关闭业务数据管道
		go func() {
			select {
			case <-ctx.Done():
				close(c)
			}
		}()
	}
	return c
}
