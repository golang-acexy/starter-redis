package test

import (
	"context"
	"fmt"
	"github.com/golang-acexy/starter-redis/redisstarter"
	"testing"
)

var key = redisstarter.RedisKey{
	KeyFormat: "key-bloom",
}

func TestReserve(t *testing.T) {
	err := redisstarter.BloomCmd().Reserve(context.Background(), key, 0.01, 10)
	if err != nil {
		t.Error(err)
	}
}

func TestInfo(t *testing.T) {
	info, err := redisstarter.BloomCmd().Info(context.Background(), key)
	if err != nil {
		t.Error(err)
	}
	t.Log(info)
}

func TestAdd(t *testing.T) {
	err := redisstarter.BloomCmd().Add(context.Background(), key, "test")
	if err != nil {
		t.Error(err)
	}
}
func TestMAdd(t *testing.T) {
	err := redisstarter.BloomCmd().MAdd(context.Background(), key, []string{"1", "2", "3"})
	if err != nil {
		t.Error(err)
	}
}

func TestBloomExists(t *testing.T) {
	fmt.Println(redisstarter.BloomCmd().Exists(context.Background(), key, "test"))
	fmt.Println(redisstarter.BloomCmd().Exists(context.Background(), key, "1"))
	fmt.Println(redisstarter.BloomCmd().Exists(context.Background(), key, "4"))
}

func TestMExists(t *testing.T) {
	fmt.Println(redisstarter.BloomCmd().MExists(context.Background(), key, []string{"test", "1", "4"}))
}
