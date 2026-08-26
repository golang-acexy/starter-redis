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
	pending      map[string]struct{}
	generation   uint64
	pubSubsMutex sync.Mutex
}

var topicCmd = &cmdTopic{
	pubSubs: make(map[string]*redis.PubSub),
	pending: make(map[string]struct{}),
}

func TopicCmd() *cmdTopic {
	return topicCmd
}

// Publish 发布消息
func (c *cmdTopic) Publish(key RedisKey, data string, keyAppend ...any) error {
	keyString := key.RawKeyString(keyAppend...)
	return redisClient.Publish(context.Background(), keyString, data).Err()
}

// Subscribe 订阅消息（独立连接）
func (c *cmdTopic) Subscribe(ctx context.Context, key RedisKey, keyAppend ...any) (<-chan *redis.Message, error) {
	keyString := key.RawKeyString(keyAppend...)

	c.pubSubsMutex.Lock()
	if _, ok := c.pubSubs[keyString]; ok {
		c.pubSubsMutex.Unlock()
		return nil, fmt.Errorf("%w: %s", ErrAlreadySubscribedToTopic, keyString)
	}
	if _, ok := c.pending[keyString]; ok {
		c.pubSubsMutex.Unlock()
		return nil, fmt.Errorf("%w: %s", ErrAlreadySubscribedToTopic, keyString)
	}
	generation := c.generation
	c.pending[keyString] = struct{}{}
	c.pubSubsMutex.Unlock()

	pubSub := redisClient.Subscribe(ctx, keyString)
	_, err := pubSub.Receive(ctx)
	if err != nil {
		_ = pubSub.Close()
		c.clearPending(keyString)
		return nil, err
	}

	c.pubSubsMutex.Lock()
	delete(c.pending, keyString)
	if generation != c.generation {
		c.pubSubsMutex.Unlock()
		_ = pubSub.Close()
		return nil, fmt.Errorf("%w: %s", ErrSubscriptionClosed, keyString)
	}
	c.pubSubs[keyString] = pubSub
	c.pubSubsMutex.Unlock()
	return pubSub.Channel(), nil
}

func (c *cmdTopic) clearPending(key string) {
	c.pubSubsMutex.Lock()
	delete(c.pending, key)
	c.pubSubsMutex.Unlock()
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
func (c *cmdTopic) Unsubscribe(key RedisKey, keyAppend ...any) error {
	keyString := key.RawKeyString(keyAppend...)
	c.pubSubsMutex.Lock()
	pubSub, ok := c.pubSubs[keyString]
	if !ok {
		c.pubSubsMutex.Unlock()
		return fmt.Errorf("%w: %s", ErrNotSubscribedToTopic, keyString)
	}
	delete(c.pubSubs, keyString)
	c.pubSubsMutex.Unlock()

	err := pubSub.Unsubscribe(context.Background(), keyString)
	_ = pubSub.Close()
	return err
}

func (c *cmdTopic) closeAll(ctx context.Context) {
	c.pubSubsMutex.Lock()
	subs := c.pubSubs
	c.pubSubs = make(map[string]*redis.PubSub)
	c.generation++
	c.pubSubsMutex.Unlock()

	for key, pubSub := range subs {
		_ = pubSub.Unsubscribe(ctx, key)
		_ = pubSub.Close()
	}
}
