package test

import (
	"context"
	"fmt"
	"github.com/golang-acexy/starter-parent/parentmodule/declaration"
	"github.com/golang-acexy/starter-redis/redismodule"
	"github.com/redis/go-redis/v9"
	"testing"
)

type User struct {
	Name string
}

func init() {
	rModule = &redismodule.RedisModule{
		RedisConfig: &redis.UniversalOptions{
			Addrs:    []string{":6379"},
			Password: "tech-acexy",
		},
	}
	moduleLoaders = []declaration.ModuleLoader{rModule}
	m := declaration.Module{ModuleLoaders: moduleLoaders}

	err := m.Load()
	if err != nil {
		fmt.Printf("%+v\n", err)
	}
}

func TestSet(t *testing.T) {

	//value := struct {
	//	Name string
	//}{Name: "张三"} // error

	//value := map[string]string{"key": "value"} // error

	//---

	//value := []byte{1, 2, 3}

	//value := "string"

	//value := 1.2

	//value := []string{"1", "2"}

	value := User{Name: "张三"}

	err := redismodule.SetAnyWithJson(context.Background(), "key", value)
	if err != nil {
		fmt.Printf("%+v\n", err)
	}

	err = redismodule.SetAny(context.Background(), "key1", 123.4)
	if err != nil {
		fmt.Printf("%+v\n", err)
	}

	err = redismodule.SetBytes(context.Background(), "key2", []byte{1, 2, 3})
	if err != nil {
		fmt.Printf("%+v\n", err)
	}

	err = redismodule.MSet(context.Background(), map[redismodule.RedisKey]string{"mkey1": "1", "mkey2": "2"})
	if err != nil {
		fmt.Printf("%+v\n", err)
	}
}

func TestGet(t *testing.T) {

	fmt.Println(redismodule.Get(context.Background(), "key2"))

	var user User // error
	fmt.Println(redismodule.GetAny(context.Background(), "key", &user))
	fmt.Printf("%+v\n", user)

	fmt.Println(redismodule.GetAnyWithJson(context.Background(), "key", &user))
	fmt.Printf("%+v\n", user)

	var floatV float64
	fmt.Println(redismodule.GetAny(context.Background(), "key1", &floatV))
	fmt.Printf("%v\n", floatV)

	fmt.Println(redismodule.MGet(context.Background(), "mkey1", "mkey2"))

	type Strings struct {
		Value1 string `redis:"mkey1"`
		Value2 string `redis:"mkey2"`
	}

	var strings Strings
	fmt.Println(redismodule.MGetAny(context.Background(), &strings, "mkey1", "mkey2"))
	fmt.Printf("%+v\n", strings)

}

func TestClient(t *testing.T) {

}
