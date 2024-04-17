package test

import (
	"context"
	"fmt"
	"github.com/golang-acexy/starter-parent/parentmodule/declaration"
	"github.com/golang-acexy/starter-redis/redismodule"
	"github.com/redis/go-redis/v9"
	"testing"
	"time"
)

var m declaration.Module

func init() {
	rModule = &redismodule.RedisModule{
		RedisConfig: redis.UniversalOptions{
			Addrs:    []string{":6379", ":6381", ":6380"},
			Password: "tech-acexy",
		},
	}
	moduleLoaders = []declaration.ModuleLoader{rModule}
	m = declaration.Module{ModuleLoaders: moduleLoaders}

	err := m.Load()
	if err != nil {
		fmt.Printf("%+v\n", err)
	}
}

func TestSet(t *testing.T) {
	stringType := redismodule.StringCmd()

	key1 := redismodule.RedisKey{
		KeyFormat: "string:%d:%s",
		Expire:    time.Second,
	}
	_ = stringType.Set(context.Background(), key1, "你好", 1, "2")
	fmt.Println(stringType.Get(context.Background(), key1, 1, "2"))
	time.Sleep(time.Second * 2)
	fmt.Println(stringType.Get(context.Background(), key1, 1, "2"))
}

type User struct {
	Name string
}

func (u *User) MarshalBinary() (data []byte, err error) {
	return []byte(u.Name), nil
}

func (u *User) UnmarshalBinary(data []byte) error {
	u.Name = string(data)
	return nil
}

type Person struct {
	Name string
}

func TestSetAny(t *testing.T) {

	stringType := redismodule.StringCmd()
	key := redismodule.RedisKey{
		KeyFormat: "key2",
	}

	value := &User{Name: "张三"}
	err := stringType.SetAny(context.Background(), key, value)
	if err != nil {
		fmt.Printf("%+v\n", err)
		return
	}

	result := User{}
	fmt.Println(stringType.GetAny(context.Background(), key, &result))
	fmt.Printf("%v\n", result)
}

func TestSetJson(t *testing.T) {
	stringType := redismodule.StringCmd()
	key := redismodule.RedisKey{
		KeyFormat: "json",
	}

	value := &Person{Name: "李四"}
	err := stringType.SetAnyWithJson(context.Background(), key, value)
	if err != nil {
		fmt.Printf("%+v\n", err)
		return
	}

	result := Person{}
	fmt.Println(stringType.GetAnyWithJson(context.Background(), key, &result))
	fmt.Printf("%v\n", result)

}

func TestMSet(t *testing.T) {
	stringType := redismodule.StringCmd()
	err := stringType.MSet(context.Background(), map[string]string{"11": "aa"})
	if err != nil {
		fmt.Printf("%+v\n", err)
		return
	}
	fmt.Println(stringType.MGet(context.Background(), "11"))

	err = stringType.MSetWithHashTag(context.Background(), "a", map[string]string{"a": "2", "b": "3"})
	if err != nil {
		fmt.Printf("%+v\n", err)
		return
	}
	fmt.Println(stringType.MGetWithHashTag(context.Background(), "a", "a", "aaaa", "b"))

	err = stringType.MSetBytes(context.Background(), map[string][]byte{"b1": {1, 2, 3}})
	if err != nil {
		fmt.Printf("%+v\n", err)
		return
	}
	fmt.Println(stringType.MGetBytes(context.Background(), "b1"))
}

func TestClient(t *testing.T) {

}
