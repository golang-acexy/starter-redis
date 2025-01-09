package test

import (
	"fmt"
	"github.com/golang-acexy/starter-redis/redisstarter"
	"testing"
	"time"
)

func TestSetCmd(t *testing.T) {
	stringType := redisstarter.StringCmd()

	key1 := redisstarter.RedisKey{
		KeyFormat: "string:%d:%s",
		Expire:    time.Second,
	}
	_ = stringType.Set(key1, "你好", 1, "2")
	fmt.Println(stringType.Get(key1, 1, "2"))
	time.Sleep(time.Second * 2)
	fmt.Println(stringType.Get(key1, 1, "2"))
}

type User struct {
	Name string
	Age  int
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

	stringType := redisstarter.StringCmd()
	key := redisstarter.RedisKey{
		KeyFormat: "key2",
	}

	value := &User{Name: "张三"}
	err := stringType.SetAny(key, value)
	if err != nil {
		fmt.Printf("%+v\n", err)
		return
	}

	result := User{}
	fmt.Println(stringType.GetAny(key, &result))
	fmt.Printf("%v\n", result)
}

func TestSetJson(t *testing.T) {
	stringType := redisstarter.StringCmd()
	key := redisstarter.RedisKey{
		KeyFormat: "json",
	}

	value := &Person{Name: "李四"}
	err := stringType.SetAnyWithJson(key, value)
	if err != nil {
		fmt.Printf("%+v\n", err)
		return
	}

	result := Person{}
	fmt.Println(stringType.GetAnyWithJson(key, &result))
	fmt.Printf("%v\n", result)

}

func TestMSet(t *testing.T) {
	stringType := redisstarter.StringCmd()
	err := stringType.MSet(map[string]string{"11": "aa"})
	if err != nil {
		fmt.Printf("%+v\n", err)
		return
	}
	fmt.Println(stringType.MGet("11"))

	err = stringType.MSetWithHashTag("a", map[string]string{"a": "2", "b": "3"})
	if err != nil {
		fmt.Printf("%+v\n", err)
		return
	}
	fmt.Println(stringType.MGetWithHashTag("a", "a", "aaaa", "b"))

	err = stringType.MSetBytes(map[string][]byte{"b1": {1, 2, 3}})
	if err != nil {
		fmt.Printf("%+v\n", err)
		return
	}
	fmt.Println(stringType.MGetBytes("b1"))
}

func TestClient(t *testing.T) {

}
