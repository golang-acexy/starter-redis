package test

import (
	"fmt"
	"github.com/golang-acexy/starter-redis/redisstarter"
	"testing"
	"time"
)

func TestHSet(t *testing.T) {
	hashType := redisstarter.HashCmd()
	key := redisstarter.RedisKey{
		KeyFormat: "key-hash",
		Expire:    time.Second * 1,
	}
	err := hashType.HSet(key, "name1", "value1")
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(hashType.HGet(key, "name1"))
	time.Sleep(time.Second * 2)
	fmt.Println(hashType.HGet(key, "name1"))
}

func TestHMSet(t *testing.T) {
	hashType := redisstarter.HashCmd()
	key := redisstarter.RedisKey{
		KeyFormat: "key-m-hash",
	}
	err := hashType.HMSet(key, map[string]string{"1": "2", "3": "4", "5": "6"})
	if err != nil {
		fmt.Println(err)
		return
	}
	result, _ := hashType.HMGet(key, []string{"1", "3", "5", "4"})
	fmt.Println(hashType.HExists(key, "7"))
	fmt.Println(result)
}

func TestHGetAll(t *testing.T) {
	hashType := redisstarter.HashCmd()
	key := redisstarter.RedisKey{
		KeyFormat: "key-m-hash",
	}
	fmt.Println(hashType.HGetAll(key))
}
