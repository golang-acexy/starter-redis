package test

import (
	"context"
	"fmt"
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
		err = cmd.Unsubscribe(context.Background(), key1)
		if err != nil {
			fmt.Println("取消订阅异常", err)
		}
	}()

	go func() {
		for i := 1; i <= 10; i++ {
			_ = cmd.Publish(context.Background(), key1, "hello")
			time.Sleep(time.Millisecond * 200)
		}
	}()

	// 循环不会退出，即使取消了订阅，但是底层通道并没有关闭
	for message := range messages1 {
		fmt.Println("接收到数据", message)
	}
}

func TestQueue(t *testing.T) {
	cmd := redisstarter.QueueCmd()
	key1 := redisstarter.RedisKey{
		KeyFormat: "queue",
	}

	key2 := redisstarter.RedisKey{
		KeyFormat: "queue1",
	}

	go func() {
		for i := 1; i <= 10; i++ {
			_ = cmd.Push(context.Background(), false, key1, fmt.Sprintf("1 hello %d", i))
			time.Sleep(time.Millisecond * 200)
			fmt.Println("1 发送数据")
		}
	}()

	ctx1, cancel1 := context.WithCancel(context.Background())

	go func() {
		time.Sleep(time.Second * 10)
		fmt.Println("1 取消接收数据")
		cancel1()
	}()

	c1 := cmd.BPop(ctx1, true, time.Second, key1)

	go func() {
		for i := 1; i <= 10; i++ {
			_ = cmd.Push(context.Background(), false, key2, fmt.Sprintf("2 hello %d", i))
			time.Sleep(time.Millisecond * 300)
			fmt.Println("2 发送数据")
		}
	}()

	ctx2, cancel2 := context.WithCancel(context.Background())

	go func() {
		time.Sleep(time.Second * 15)
		fmt.Println("2 取消接收数据")
		cancel2()
	}()

	c2 := cmd.BPop(ctx2, true, time.Second, key2)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
	F1:
		for {
			select {
			case <-ctx1.Done():
				fmt.Println("1 done")
				wg.Done()
				break F1
			case data := <-c1:
				fmt.Println("1 接收到数据", data)
			}
		}
	}()

	go func() {
	F2:
		for {
			select {
			case <-ctx2.Done():
				fmt.Println("2 done")
				wg.Done()
				break F2
			case data := <-c2:
				fmt.Println("2 接收到数据", data)
			}
		}

	}()
	wg.Wait()
}

func TestQueuePop(t *testing.T) {
	cmd := redisstarter.QueueCmd()
	key := redisstarter.RedisKey{
		KeyFormat: "queue1",
	}
	var wait sync.WaitGroup
	wait.Add(2)
	ctx, cancelFunc := context.WithCancel(context.Background())
	c := cmd.BPop(ctx, true, 0, key)
	go func() {
		for d := range c {
			fmt.Println("work1 获取到数据", d)
			time.Sleep(time.Second * 2)
		}
		fmt.Println("1数据管道已关闭")
		wait.Done()
	}()

	go func() {
		for d := range c {
			fmt.Println("work2 获取到数据", d)
			time.Sleep(time.Second)
		}
		fmt.Println("2数据管道已关闭")
		wait.Done()
	}()

	go func() {
		time.Sleep(time.Second * 3)
		cancelFunc()
		fmt.Println("取消监听")
	}()

	wait.Wait()
}
