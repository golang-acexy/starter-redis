package test

import (
	"context"
	"fmt"
	"github.com/golang-acexy/starter-redis/redismodule"
	"testing"
	"time"
)

func TestDel(t *testing.T) {
	keyType := redismodule.KeyCmd()
	key := redismodule.RedisKey{
		KeyFormat: "key-hash",
	}
	fmt.Println(redismodule.OriginKeyString(key.KeyFormat))
	fmt.Println(keyType.Del(context.Background(), key))
}

func TestExists(t *testing.T) {
	keyType := redismodule.KeyCmd()
	key := redismodule.RedisKey{
		KeyFormat: "key-hash",
	}
	fmt.Println(keyType.Exists(context.Background(), key))
}

func TestExpire(t *testing.T) {
	keyType := redismodule.KeyCmd()
	key := redismodule.RedisKey{
		KeyFormat: "key-hash",
	}
	fmt.Println(keyType.Expire(context.Background(), redismodule.OriginKeyString(key.KeyFormat), time.Second*5))
}
