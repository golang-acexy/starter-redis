package test

import (
	"fmt"
	"github.com/acexy/golang-toolkit/math/random"
	"github.com/acexy/golang-toolkit/util/json"
	"github.com/golang-acexy/starter-parent/parent"
	"github.com/golang-acexy/starter-redis/redisstarter"
	"github.com/redis/go-redis/v9"
	"testing"
	"time"
)

const isCluster = false

var loader *parent.StarterLoader

var standalone = &redisstarter.RedisStarter{
	RedisConfig: redis.UniversalOptions{
		Addrs:    []string{":6379"},
		Password: "tech-acexy",
		DB:       0,
	},
}

var cluster = &redisstarter.RedisStarter{
	RedisConfig: redis.UniversalOptions{
		Addrs:    []string{":6379", ":6381", ":6380"},
		Password: "tech-acexy",
	},
	InitFunc: func(instance redis.UniversalClient) {
		fmt.Println(instance.PoolStats())
	},
}

func TestMain(m *testing.M) {

	var loadType *redisstarter.RedisStarter
	if isCluster {
		loadType = cluster
	} else {
		loadType = standalone
	}
	loader = parent.NewStarterLoader([]parent.Starter{loadType})

	err := loader.Start()
	if err != nil {
		fmt.Printf("%+v\n", err)
		return
	}
	m.Run()
}

// 单实例Redis
func TestStandalone(t *testing.T) {

	// 启动一批协程，模拟并发多连接执行中场景
	go func() {
		for i := 1; i <= 5; i++ {
			go func() {
				for {
					err := redisstarter.StringCmd().Set(redisstarter.RedisKey{KeyFormat: random.RandString(5), Expire: time.Second * 10}, random.RandString(5))
					if err != nil {
						fmt.Println(err)
					}
					time.Sleep(time.Millisecond * 200)
				}
			}()
		}
	}()
	time.Sleep(time.Second * 10)
	stopResult, err := loader.StopBySetting()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(json.ToJsonFormat(stopResult))
}

// 集群Redis
func TestCluster(t *testing.T) {
	clusterLoader := parent.NewStarterLoader([]parent.Starter{cluster})
	err := clusterLoader.Start()
	if err != nil {
		fmt.Printf("%+v\n", err)
		return
	}
	// 启动一批协程，模拟并发多连接执行中场景
	go func() {
		for i := 1; i <= 10; i++ {
			go func() {
				for {
					err = redisstarter.StringCmd().Set(redisstarter.RedisKey{KeyFormat: random.RandString(5), Expire: time.Second * 10}, random.RandString(5))
					if err != nil {
						fmt.Println("invoke err", err)
					}
				}
			}()
		}
	}()

	time.Sleep(time.Second * 10)
	stopResult, err := loader.StopBySetting()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(json.ToJsonFormat(stopResult))
}
