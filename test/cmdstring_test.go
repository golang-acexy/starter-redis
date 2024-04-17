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
	stringType := redismodule.StringType()

	key1 := redismodule.RedisKey{
		KeyFormat: "string:%d:%s",
		Expire:    time.Second,
	}
	_ = stringType.Set(context.Background(), key1, "你好", 1, "2")
	fmt.Println(stringType.Get(context.Background(), key1, 1, "2"))
	time.Sleep(time.Second * 2)
	fmt.Println(stringType.Get(context.Background(), key1, 1, "2"))

	//	err := redismodule.SetAnyWithJson(context.Background(), "key", value)
	//	if err != nil {
	//		fmt.Printf("%+v\n", err)
	//	}
	//
	//	err = redismodule.SetAny(context.Background(), "key1", 123.4)
	//	if err != nil {
	//		fmt.Printf("%+v\n", err)
	//	}
	//
	//	err = redismodule.SetBytes(context.Background(), "key2", []byte{1, 2, 3})
	//	if err != nil {
	//		fmt.Printf("%+v\n", err)
	//	}
	//
	//	err = redismodule.MSet(context.Background(), map[redismodule.RedisKey]string{"mkey1": "1", "mkey2": "2"})
	//	if err != nil {
	//		fmt.Printf("%+v\n", err)
	//	}
	//	fmt.Println(m.UnloadByConfig())
	//}
	//
	//func TestGet(t *testing.T) {
	//
	//	fmt.Println(redismodule.Get(context.Background(), "key2"))
	//
	//	var user User // error
	//	fmt.Println(redismodule.GetAny(context.Background(), "key", &user))
	//	fmt.Printf("%+v\n", user)
	//
	//	fmt.Println(redismodule.GetAnyWithJson(context.Background(), "key", &user))
	//	fmt.Printf("%+v\n", user)
	//
	//	var floatV float64
	//	fmt.Println(redismodule.GetAny(context.Background(), "key1", &floatV))
	//	fmt.Printf("%v\n", floatV)
	//
	//	fmt.Println(redismodule.MGet(context.Background(), "mkey1", "mkey2"))
	//
	//	type Strings struct {
	//		Value1 string `redis:"mkey1"`
	//		Value2 string `redis:"mkey2"`
	//	}
	//
	//	var strings Strings
	//	fmt.Println(redismodule.MGetAny(context.Background(), &strings, "mkey1", "mkey2"))
	//	fmt.Printf("%+v\n", strings)
	//	fmt.Println(m.Unload(2))
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

func TestSetAny(t *testing.T) {

	stringType := redismodule.StringType()
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

func TestMSet(t *testing.T) {
	stringType := redismodule.StringType()
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
}

func TestClient(t *testing.T) {

}
