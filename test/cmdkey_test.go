package test

import (
	"fmt"
	"github.com/golang-acexy/starter-redis/redisstarter"
	"testing"
	"time"
)

func TestDel(t *testing.T) {
	keyType := redisstarter.KeyCmd()
	key := redisstarter.RedisKey{
		KeyFormat: "key-hash",
	}
	fmt.Println(redisstarter.OriginKeyString(key.KeyFormat))
	fmt.Println(keyType.Del(key))
}

func TestExists(t *testing.T) {
	keyType := redisstarter.KeyCmd()
	key := redisstarter.RedisKey{
		KeyFormat: "key-hash",
	}
	fmt.Println(keyType.Exists(key))
}

func TestExpire(t *testing.T) {
	keyType := redisstarter.KeyCmd()
	key := redisstarter.RedisKey{
		KeyFormat: "key-hash",
	}
	fmt.Println(keyType.Expire(key, time.Second*5))
}
