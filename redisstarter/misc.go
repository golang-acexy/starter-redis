package redisstarter

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type cmdTopic struct {
	pubSubs          map[string]*topicSubscription
	pending          map[string]*topicPending
	generation       uint64
	nextSubscriberID uint64
	pubSubsMutex     sync.Mutex
}

// topicSubscription 表示一个 Redis topic 的底层订阅及其本地订阅者。
// 同一 topic 只建立一个 Redis PubSub 连接，收到的消息会扇出给每个本地订阅者。
type topicSubscription struct {
	pubSub      *redis.PubSub
	subscribers map[uint64]*topicSubscriber
}

// topicPending 表示正在建立的底层订阅。重复订阅会等待该订阅完成，避免并发创建连接。
type topicPending struct {
	ready chan struct{}
	err   error
}

// topicSubscriber 是一个本地订阅者。messages 关闭表示订阅已取消或底层订阅已结束。
type topicSubscriber struct {
	messages chan *redis.Message
	done     chan struct{}
	mutex    sync.Mutex
	closed   bool
}

const topicSubscriberChannelSize = 100

var topicCmd = &cmdTopic{
	pubSubs: make(map[string]*topicSubscription),
	pending: make(map[string]*topicPending),
}

func TopicCmd() *cmdTopic {
	return topicCmd
}

// Publish 发布消息
func (c *cmdTopic) Publish(key RedisKey, data string, keyAppend ...any) error {
	keyString := key.RawKeyString(keyAppend...)
	return rawRedisClient().Publish(context.Background(), keyString, data).Err()
}

// Subscribe 订阅消息。每个 topic 仅允许一次基础订阅；需要多个本地订阅者时使用 SubscribeRepeat。
func (c *cmdTopic) Subscribe(ctx context.Context, key RedisKey, keyAppend ...any) (<-chan *redis.Message, error) {
	messages, _, err := c.subscribe(ctx, key, false, keyAppend...)
	return messages, err
}

// SubscribeRepeat 重复订阅同一个 topic，并返回独立的本地消息通道和取消函数。
//
// 同一 topic 的所有重复订阅共享一个底层 Redis PubSub 连接，但每个调用方都会收到
// 独立通道，因此消息不会被多个调用方竞争消费。取消函数只取消本次调用创建的订阅，
// 可重复调用；当最后一个本地订阅者取消时，底层 Redis 订阅会一并关闭。
//
// 本地通道使用有限缓冲。Redis Pub/Sub 本身是至多一次消息模型；当某个订阅者持续
// 消费过慢、其缓冲区已满时，新的消息会仅对该订阅者丢弃，避免阻塞其他订阅者。
func (c *cmdTopic) SubscribeRepeat(ctx context.Context, key RedisKey, keyAppend ...any) (<-chan *redis.Message, func(), error) {
	return c.subscribe(ctx, key, true, keyAppend...)
}

func (c *cmdTopic) subscribe(ctx context.Context, key RedisKey, allowRepeat bool, keyAppend ...any) (<-chan *redis.Message, func(), error) {
	keyString := key.RawKeyString(keyAppend...)

	for {
		c.pubSubsMutex.Lock()
		if subscription, ok := c.pubSubs[keyString]; ok {
			if !allowRepeat {
				c.pubSubsMutex.Unlock()
				return nil, nil, fmt.Errorf("%w: %s", ErrAlreadySubscribedToTopic, keyString)
			}
			subscriberID, subscriber := c.addSubscriberLocked(subscription)
			c.pubSubsMutex.Unlock()
			c.watchSubscriberContext(ctx, keyString, subscriberID, subscriber)
			return subscriber.messages, c.unsubscribeRepeat(keyString, subscriberID), nil
		}
		if pending, ok := c.pending[keyString]; ok {
			if !allowRepeat {
				c.pubSubsMutex.Unlock()
				return nil, nil, fmt.Errorf("%w: %s", ErrAlreadySubscribedToTopic, keyString)
			}
			c.pubSubsMutex.Unlock()
			select {
			case <-ctx.Done():
				return nil, nil, ctx.Err()
			case <-pending.ready:
				if pending.err != nil {
					return nil, nil, pending.err
				}
				continue
			}
		}

		generation := c.generation
		pending := &topicPending{ready: make(chan struct{})}
		c.pending[keyString] = pending
		c.pubSubsMutex.Unlock()

		client := rawRedisClient()
		if client == nil {
			c.completePending(keyString, pending, ErrRedisClientNotStarted)
			return nil, nil, ErrRedisClientNotStarted
		}
		pubSub := client.Subscribe(ctx, keyString)
		_, err := pubSub.Receive(ctx)
		if err != nil {
			_ = pubSub.Close()
			c.completePending(keyString, pending, err)
			return nil, nil, err
		}

		c.pubSubsMutex.Lock()
		if generation != c.generation {
			c.pubSubsMutex.Unlock()
			_ = pubSub.Close()
			err = fmt.Errorf("%w: %s", ErrSubscriptionClosed, keyString)
			c.completePending(keyString, pending, err)
			return nil, nil, err
		}
		subscription := &topicSubscription{
			pubSub:      pubSub,
			subscribers: make(map[uint64]*topicSubscriber),
		}
		c.pubSubs[keyString] = subscription
		subscriberID, subscriber := c.addSubscriberLocked(subscription)
		c.pubSubsMutex.Unlock()
		c.completePending(keyString, pending, nil)

		go c.forwardMessages(keyString, subscription)
		c.watchSubscriberContext(ctx, keyString, subscriberID, subscriber)
		return subscriber.messages, c.unsubscribeRepeat(keyString, subscriberID), nil
	}
}

func (c *cmdTopic) addSubscriberLocked(subscription *topicSubscription) (uint64, *topicSubscriber) {
	c.nextSubscriberID++
	subscriber := &topicSubscriber{
		messages: make(chan *redis.Message, topicSubscriberChannelSize),
		done:     make(chan struct{}),
	}
	subscription.subscribers[c.nextSubscriberID] = subscriber
	return c.nextSubscriberID, subscriber
}

func (c *cmdTopic) completePending(key string, pending *topicPending, err error) {
	c.pubSubsMutex.Lock()
	if current, ok := c.pending[key]; ok && current == pending {
		delete(c.pending, key)
		pending.err = err
		close(pending.ready)
	}
	c.pubSubsMutex.Unlock()
}

func (c *cmdTopic) watchSubscriberContext(ctx context.Context, key string, subscriberID uint64, subscriber *topicSubscriber) {
	go func() {
		select {
		case <-ctx.Done():
			c.unsubscribeSubscriber(key, subscriberID)
		case <-subscriber.done:
		}
	}()
}

// SubscribeRetry 订阅消息（重试连接）
func (c *cmdTopic) SubscribeRetry(ctx context.Context, topicKey RedisKey, handle func(*redis.Message)) {
	if handle == nil {
		return
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			ch, cancel, err := c.SubscribeRepeat(ctx, topicKey)
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
				handle(msg)
			}
			cancel()
		}
	}()
}

// Unsubscribe 取消一个 topic 的全部本地订阅并释放底层 Redis 连接。
// 若只需要取消 SubscribeRepeat 创建的某一个订阅，请调用其返回的取消函数。
func (c *cmdTopic) Unsubscribe(key RedisKey, keyAppend ...any) error {
	keyString := key.RawKeyString(keyAppend...)
	subscription, subscribers := c.detachSubscription(keyString, nil)
	if subscription == nil {
		return fmt.Errorf("%w: %s", ErrNotSubscribedToTopic, keyString)
	}
	return closeTopicSubscription(context.Background(), keyString, subscription, subscribers)
}

func (c *cmdTopic) unsubscribeRepeat(key string, subscriberID uint64) func() {
	return func() {
		c.unsubscribeSubscriber(key, subscriberID)
	}
}

func (c *cmdTopic) unsubscribeSubscriber(key string, subscriberID uint64) {
	c.pubSubsMutex.Lock()
	subscription, ok := c.pubSubs[key]
	if !ok {
		c.pubSubsMutex.Unlock()
		return
	}
	subscriber, ok := subscription.subscribers[subscriberID]
	if !ok {
		c.pubSubsMutex.Unlock()
		return
	}
	delete(subscription.subscribers, subscriberID)
	closeSubscription := len(subscription.subscribers) == 0
	if closeSubscription {
		delete(c.pubSubs, key)
	}
	c.pubSubsMutex.Unlock()

	subscriber.close()
	if closeSubscription {
		_ = closeTopicSubscription(context.Background(), key, subscription, nil)
	}
}

func (c *cmdTopic) closeAll(ctx context.Context) {
	c.pubSubsMutex.Lock()
	subs := make(map[string]*topicSubscription, len(c.pubSubs))
	for key, subscription := range c.pubSubs {
		subs[key] = subscription
	}
	c.pubSubs = make(map[string]*topicSubscription)
	c.generation++
	for key, pending := range c.pending {
		delete(c.pending, key)
		pending.err = fmt.Errorf("%w: %s", ErrSubscriptionClosed, key)
		close(pending.ready)
	}
	c.pubSubsMutex.Unlock()

	for key, subscription := range subs {
		_ = closeTopicSubscription(ctx, key, subscription, c.takeSubscribers(subscription))
	}
}

func (c *cmdTopic) forwardMessages(key string, subscription *topicSubscription) {
	for message := range subscription.pubSub.Channel() {
		if !c.deliverMessage(key, subscription, message) {
			return
		}
	}

	subscription, subscribers := c.detachSubscription(key, subscription)
	if subscription != nil {
		_ = closeTopicSubscription(context.Background(), key, subscription, subscribers)
	}
}

func (c *cmdTopic) deliverMessage(key string, subscription *topicSubscription, message *redis.Message) bool {
	c.pubSubsMutex.Lock()
	if c.pubSubs[key] != subscription {
		c.pubSubsMutex.Unlock()
		return false
	}
	subscribers := make([]*topicSubscriber, 0, len(subscription.subscribers))
	for _, subscriber := range subscription.subscribers {
		subscribers = append(subscribers, subscriber)
	}
	c.pubSubsMutex.Unlock()

	for _, subscriber := range subscribers {
		subscriber.deliver(message)
	}
	return true
}

func (c *cmdTopic) detachSubscription(key string, expected *topicSubscription) (*topicSubscription, []*topicSubscriber) {
	c.pubSubsMutex.Lock()
	defer c.pubSubsMutex.Unlock()
	subscription, ok := c.pubSubs[key]
	if !ok || (expected != nil && subscription != expected) {
		return nil, nil
	}
	delete(c.pubSubs, key)
	return subscription, c.takeSubscribers(subscription)
}

func (c *cmdTopic) takeSubscribers(subscription *topicSubscription) []*topicSubscriber {
	subscribers := make([]*topicSubscriber, 0, len(subscription.subscribers))
	for _, subscriber := range subscription.subscribers {
		subscribers = append(subscribers, subscriber)
	}
	subscription.subscribers = make(map[uint64]*topicSubscriber)
	return subscribers
}

func closeTopicSubscription(ctx context.Context, key string, subscription *topicSubscription, subscribers []*topicSubscriber) error {
	for _, subscriber := range subscribers {
		subscriber.close()
	}
	if subscription.pubSub == nil {
		return nil
	}
	err := subscription.pubSub.Unsubscribe(ctx, key)
	_ = subscription.pubSub.Close()
	return err
}

func (s *topicSubscriber) deliver(message *redis.Message) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.closed {
		return
	}
	select {
	case s.messages <- message:
	default:
	}
}

func (s *topicSubscriber) close() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.closed {
		return
	}
	s.closed = true
	close(s.done)
	close(s.messages)
}
