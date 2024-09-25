package redisstarter

import (
	"context"
	"errors"
	"github.com/acexy/golang-toolkit/logger"
	"github.com/redis/go-redis/v9"
	"sync"
	"time"
)

type cmdTopic struct {
	pubSub      *redis.PubSub
	pubSubMutex sync.Mutex
}

var topicCmd = &cmdTopic{}

func TopicCmd() *cmdTopic {
	return topicCmd
}

// Publish 发送消息
func (c *cmdTopic) Publish(ctx context.Context, key RedisKey, data string, keyAppend ...interface{}) error {
	keyString := OriginKeyString(key.KeyFormat, keyAppend...)
	return redisClient.Publish(ctx, keyString, data).Err()
}

// Subscribe 订阅消费Topic
func (c *cmdTopic) Subscribe(ctx context.Context, key RedisKey, keyAppend ...interface{}) (<-chan *redis.Message, error) {
	keyString := OriginKeyString(key.KeyFormat, keyAppend...)
	pubSub := redisClient.Subscribe(ctx, keyString)
	_, err := pubSub.Receive(ctx)
	if err != nil {
		return nil, err
	}
	defer c.pubSubMutex.Unlock()
	c.pubSubMutex.Lock()
	if c.pubSub == nil {
		c.pubSub = pubSub
	}
	messages := pubSub.Channel()
	return messages, nil
}

// Unsubscribe 取消订阅Topic
func (c *cmdTopic) Unsubscribe(ctx context.Context, key RedisKey, keyAppend ...interface{}) error {
	keyString := OriginKeyString(key.KeyFormat, keyAppend...)
	defer c.pubSubMutex.Unlock()
	c.pubSubMutex.Lock()
	if c.pubSub == nil {
		return errors.New("unknown topic " + keyString)
	}
	return c.pubSub.Unsubscribe(ctx, keyString)
}

// list 队列
type cmdQueue struct {
}

var queueCmd = new(cmdQueue)

func QueueCmd() *cmdQueue {
	return queueCmd
}

// Push 数据入队
func (*cmdQueue) Push(ctx context.Context, directionRight bool, key RedisKey, data string, keyAppend ...interface{}) error {
	if directionRight {
		return redisClient.RPush(ctx, OriginKeyString(key.KeyFormat, keyAppend...), data).Err()
	}
	return redisClient.LPush(ctx, OriginKeyString(key.KeyFormat, keyAppend...), data).Err()
}

// BPop 数据出队
// directionRight: true 从右出，false 从左出
// timeout: 向队列获取数据的最大等待时间，0 为永久阻塞
func (*cmdQueue) BPop(ctx context.Context, directionRight bool, timeout time.Duration, key RedisKey, keyAppend ...interface{}) <-chan string {
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
