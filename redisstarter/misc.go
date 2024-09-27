package redisstarter

import (
	"context"
	"errors"
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
func (c *cmdTopic) Publish(key RedisKey, data string, keyAppend ...interface{}) error {
	keyString := OriginKeyString(key.KeyFormat, keyAppend...)
	return redisClient.Publish(context.Background(), keyString, data).Err()
}

// Subscribe 订阅消费Topic
func (c *cmdTopic) Subscribe(ctx context.Context, key RedisKey, keyAppend ...interface{}) (<-chan *redis.Message, error) {
	keyString := OriginKeyString(key.KeyFormat, keyAppend...)
	pubSub := redisClient.Subscribe(context.Background(), keyString)
	_, err := pubSub.Receive(context.Background())
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
func (c *cmdTopic) Unsubscribe(key RedisKey, keyAppend ...interface{}) error {
	keyString := OriginKeyString(key.KeyFormat, keyAppend...)
	defer c.pubSubMutex.Unlock()
	c.pubSubMutex.Lock()
	if c.pubSub == nil {
		return errors.New("unknown topic " + keyString)
	}
	return c.pubSub.Unsubscribe(context.Background(), keyString)
}
