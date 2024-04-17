package test

import (
	"context"
	"fmt"
	"github.com/golang-acexy/starter-redis/redismodule"
	"testing"
	"time"
)

func TestHSet(t *testing.T) {
	hashType := redismodule.HashCmd()
	key := redismodule.RedisKey{
		KeyFormat: "key-hash",
		Expire:    time.Second * 1,
	}
	err := hashType.HSet(context.Background(), key, "name1", "value1")
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(hashType.HGet(context.Background(), key, "name1"))
	time.Sleep(time.Second * 2)
	fmt.Println(hashType.HGet(context.Background(), key, "name1"))
}

func TestHMSet(t *testing.T) {
	hashType := redismodule.HashCmd()
	key := redismodule.RedisKey{
		KeyFormat: "key-m-hash",
	}
	err := hashType.HMSet(context.Background(), key, map[string]string{"1": "2", "3": "4", "5": "6"})
	if err != nil {
		fmt.Println(err)
		return
	}
	result, _ := hashType.HMGet(context.Background(), key, []string{"1", "3", "5", "4"})
	fmt.Println(result)
}

func TestHGetAll(t *testing.T) {
	hashType := redismodule.HashCmd()
	key := redismodule.RedisKey{
		KeyFormat: "key-m-hash",
	}
	fmt.Println(hashType.HGetAll(context.Background(), key))
}

func TestHSetJson(t *testing.T) {
	hashType := redismodule.HashCmd()
	key := redismodule.RedisKey{
		KeyFormat: "key-json-hash",
	}

	u := User{Name: "王五"}
	err := hashType.HSetAnyWithJson(context.Background(), key, "u1", u)
	if err != nil {
		fmt.Println(err)
		return
	}
	var u1 User
	fmt.Println(hashType.HGetAnyWithJson(context.Background(), key, "u1", &u1))
	fmt.Printf("%+v\n", u1)
}
