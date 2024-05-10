package test

import (
	"context"
	"fmt"
	"github.com/golang-acexy/starter-redis/redismodule"
	"testing"
)

var setCmd = redismodule.SetCmd()

func TestSAdd(t *testing.T) {
	key := redismodule.RedisKey{
		KeyFormat: "key-set",
	}
	err := setCmd.SAdd(context.Background(), key, []string{"你", "好"})
	if err != nil {
		t.Error(err)
	}
	fmt.Println(setCmd.SMembers(context.Background(), key))
}
