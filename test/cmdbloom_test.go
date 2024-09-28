package test

import (
	"fmt"
	"github.com/golang-acexy/starter-redis/redisstarter"
	"testing"
)

var key = redisstarter.RedisKey{
	KeyFormat: "key-bloom",
}

func TestReserve(t *testing.T) {
	err := redisstarter.BloomCmd().Reserve(key, 0.01, 10)
	if err != nil {
		t.Error(err)
	}
}

func TestInfo(t *testing.T) {
	info, err := redisstarter.BloomCmd().Info(key)
	if err != nil {
		t.Error(err)
	}
	t.Log(info)
}

func TestAdd(t *testing.T) {
	err := redisstarter.BloomCmd().Add(key, "test")
	if err != nil {
		t.Error(err)
	}
}
func TestMAdd(t *testing.T) {
	err := redisstarter.BloomCmd().MAdd(key, []string{"1", "2", "3"})
	if err != nil {
		t.Error(err)
	}
}

func TestBloomExists(t *testing.T) {
	fmt.Println(redisstarter.BloomCmd().Exists(key, "test"))
	fmt.Println(redisstarter.BloomCmd().Exists(key, "1"))
	fmt.Println(redisstarter.BloomCmd().Exists(key, "4"))
}

func TestMExists(t *testing.T) {
	fmt.Println(redisstarter.BloomCmd().MExists(key, []string{"test", "1", "4"}))
}
