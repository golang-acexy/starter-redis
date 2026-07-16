package redisstarter

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
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
		return nil, fmt.Errorf("%w: %s", ErrAlreadySubscribedToTopic, keyString)
	}

	pubSub := redisClient.Subscribe(ctx, keyString)
	_, err := pubSub.Receive(ctx)
	if err != nil {
		_ = pubSub.Close() // 确保接收失败时关闭连接
		return nil, err
	}
	c.pubSubs[keyString] = pubSub
	return pubSub.Channel(), nil
}

// SubscribeRetry 订阅消息（重试连接）
func (c *cmdTopic) SubscribeRetry(ctx context.Context, topicKey RedisKey, handle func(*redis.Message)) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			ch, err := c.Subscribe(ctx, topicKey)
			if err != nil {
				// 订阅失败，等待重试
				select {
				case <-ctx.Done():
					return
				case <-time.After(5 * time.Second):
				}
				continue
			}
			for msg := range ch {
				if msg == nil {
					break // channel 被关闭，跳出重连
				}
				handle(msg)
			}
			// 清理原订阅
			_ = c.Unsubscribe(topicKey)
		}
	}()
}

// Unsubscribe 取消订阅并释放连接
func (c *cmdTopic) Unsubscribe(key RedisKey, keyAppend ...interface{}) error {
	keyString := key.RawKeyString(keyAppend...)
	c.pubSubsMutex.Lock()
	defer c.pubSubsMutex.Unlock()
	pubSub, ok := c.pubSubs[keyString]
	if !ok {
		return fmt.Errorf("%w: %s", ErrNotSubscribedToTopic, keyString)
	}
	err := pubSub.Unsubscribe(context.Background(), keyString)
	_ = pubSub.Close()
	delete(c.pubSubs, keyString)
	return err
}

func (c *cmdTopic) closeAll(ctx context.Context) {
	c.pubSubsMutex.Lock()
	subs := c.pubSubs
	c.pubSubs = make(map[string]*redis.PubSub)
	c.pubSubsMutex.Unlock()

	for key, pubSub := range subs {
		_ = pubSub.Unsubscribe(ctx, key)
		_ = pubSub.Close()
	}
}
