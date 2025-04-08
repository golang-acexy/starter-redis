package test

import (
	"context"
	"fmt"
	"github.com/acexy/golang-toolkit/sys"
	"github.com/golang-acexy/starter-redis/redisstarter"
	"sync"
	"testing"
	"time"
)

func TestSubscribeTopicStopWithContext(t *testing.T) {
	cmd := redisstarter.TopicCmd()
	key1 := redisstarter.RedisKey{
		KeyFormat: "topic",
	}
	key2 := redisstarter.RedisKey{
		KeyFormat: "topic2",
	}

	ctx1, cancel1 := context.WithCancel(context.Background())

	go func() {
		time.Sleep(time.Second * 10)
		fmt.Println("通过上下文取消订阅1")
		cancel1()
	}()

	messages1, err := cmd.Subscribe(ctx1, key1)
	if err != nil {
		fmt.Println(err)
		return
	}

	ctx2, cancel2 := context.WithCancel(context.Background())

	go func() {
		time.Sleep(time.Second * 15)
		fmt.Println("通过上下文取消订阅2")
		cancel2()
	}()

	messages2, err := cmd.Subscribe(ctx2, key2)
	if err != nil {
		fmt.Println(err)
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		for {
			select {
			case <-ctx1.Done():
				fmt.Println("已取消订阅 1")
				wg.Done()
				return
			case message := <-messages1:
				fmt.Println("接收到数据", message)
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ctx2.Done():
				fmt.Println("已取消订阅 2")
				wg.Done()
				return
			case message := <-messages2:
				fmt.Println("接收到数据", message)
			}
		}
	}()

	wg.Wait()
	fmt.Println("所有订阅均取消")
}

func TestSubscribeClose(t *testing.T) {
	cmd := redisstarter.TopicCmd()
	key1 := redisstarter.RedisKey{
		KeyFormat: "topic",
	}
	_, err := cmd.Subscribe(context.Background(), key1)
	if err != nil {
		fmt.Println(err)
		return
	}
	time.Sleep(time.Second)
	loader.StopBySetting()
	sys.ShutdownHolding()
}

func TestSubscribeTopicStopWithUnsubscribe(t *testing.T) {

	cmd := redisstarter.TopicCmd()
	key1 := redisstarter.RedisKey{
		KeyFormat: "topic",
	}

	messages1, err := cmd.Subscribe(context.Background(), key1)
	if err != nil {
		fmt.Println(err)
		return
	}

	go func() {
		time.Sleep(time.Second * 10)
		fmt.Println("执行取消订阅")
		err = cmd.Unsubscribe(key1)
		if err != nil {
			fmt.Println("取消订阅异常", err)
		}
	}()

	go func() {
		for i := 1; i <= 10; i++ {
			_ = cmd.Publish(key1, "hello")
			time.Sleep(time.Millisecond * 200)
		}
	}()

	// 循环不会退出，即使取消了订阅，但是底层通道并没有关闭
	for message := range messages1 {
		fmt.Println("接收到数据", message)
	}
}
