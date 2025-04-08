package redisstarter

import (
	"context"
	"errors"
	"github.com/redis/go-redis/v9"
	"sync"
	"time"
)

type cmdTopic struct {
	pubSubs      map[string]*redis.PubSub
	pubSubsMutex sync.Mutex
}

var topicCmd = &cmdTopic{
	pubSubs: make(map[string]*redis.PubSub),
}

func TopicCmd() *cmdTopic {
	return topicCmd
}

// Publish 发布消息
func (c *cmdTopic) Publish(key RedisKey, data string, keyAppend ...interface{}) error {
	keyString := key.RawKeyString(keyAppend...)
	return redisClient.Publish(context.Background(), keyString, data).Err()
}

// Subscribe 订阅消息（独立连接）
func (c *cmdTopic) Subscribe(ctx context.Context, key RedisKey, keyAppend ...interface{}) (<-chan *redis.Message, error) {
	keyString := key.RawKeyString(keyAppend...)

	c.pubSubsMutex.Lock()
	defer c.pubSubsMutex.Unlock()

	if _, ok := c.pubSubs[keyString]; ok {
		return nil, errors.New("already subscribed to topic: " + keyString)
	}

	pubSub := redisClient.Subscribe(ctx, keyString)
	_, err := pubSub.Receive(ctx)
	if err != nil {
		_ = pubSub.Close() // 确保接收失败时关闭连接
		return nil, err
	}
	println(pubSub)
	c.pubSubs[keyString] = pubSub
	return pubSub.Channel(), nil
}

// SubscribeRetry 订阅消息（重试连接）
func SubscribeRetry(ctx context.Context, topicKey RedisKey, handle func(*redis.Message)) {
	for {
		ch, err := TopicCmd().Subscribe(ctx, topicKey)
		if err != nil {
			// 订阅失败，等待重试
			time.Sleep(2 * time.Second)
			continue
		}
		for msg := range ch {
			if msg == nil {
				break // channel 被关闭，跳出重连
			}
			handle(msg)
		}
		// 清理原订阅
		_ = TopicCmd().Unsubscribe(topicKey)
	}
}

// Unsubscribe 取消订阅并释放连接
func (c *cmdTopic) Unsubscribe(key RedisKey, keyAppend ...interface{}) error {
	keyString := key.RawKeyString(keyAppend...)
	c.pubSubsMutex.Lock()
	defer c.pubSubsMutex.Unlock()
	pubSub, ok := c.pubSubs[keyString]
	if !ok {
		return errors.New("not subscribed to topic: " + keyString)
	}
	err := pubSub.Unsubscribe(context.Background(), keyString)
	_ = pubSub.Close()
	delete(c.pubSubs, keyString)
	return err
}
