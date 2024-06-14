package redismodule

import (
	"context"
	"errors"
	"github.com/acexy/golang-toolkit/logger"
	"github.com/redis/go-redis/v9"
	"sync"
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

// FIFO
type cmdQueue struct {
}

var queueCmd = new(cmdQueue)

func QueueCmd() *cmdQueue {
	return queueCmd
}

// Push 数据入队
func (*cmdQueue) Push(ctx context.Context, key RedisKey, data string, keyAppend ...interface{}) error {
	return redisClient.LPush(ctx, OriginKeyString(key.KeyFormat, keyAppend...), data).Err()
}

// Pop 数据出队 FIFO
func (*cmdQueue) Pop(ctx context.Context, key RedisKey, keyAppend ...interface{}) <-chan string {
	keyString := OriginKeyString(key.KeyFormat, keyAppend...)
	c := make(chan string)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				data, err := redisClient.BRPop(ctx, 0, keyString).Result()
				if err != nil {
					logger.Logrus().Error("pop data error", keyString, err)
				} else {
					c <- data[1]
				}
			}
		}
	}()
	return c
}
