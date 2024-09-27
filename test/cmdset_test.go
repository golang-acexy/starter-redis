package test

import (
	"fmt"
	"github.com/acexy/golang-toolkit/util/json"
	"github.com/golang-acexy/starter-redis/redisstarter"
	"testing"
)

var setCmd = redisstarter.SetCmd()

type ObjectStruct struct {
	Name string
	Age  int
}

func (s *ObjectStruct) UnmarshalBinary(data []byte) error {
	return json.ParseBytesError(data, &s)
}

func (s *ObjectStruct) MarshalBinary() (data []byte, err error) {
	return json.ToJsonBytesError(s)
}

func TestSet(t *testing.T) {
	key := redisstarter.RedisKey{
		KeyFormat: "key-set",
	}
	redisstarter.KeyCmd().Del(key)
	err := setCmd.SAdds(key, []interface{}{
		"你",
		"好",
		&ObjectStruct{
			Name: "name",
			Age:  18,
		},
	})
	if err != nil {
		t.Error(err)
	}
	fmt.Println(setCmd.SMembers(key))
	fmt.Println(setCmd.SMembersMap(key))
}

func TestSetStruct(t *testing.T) {
	key := redisstarter.RedisKey{
		KeyFormat: "key-set-struct",
	}
	redisstarter.KeyCmd().Del(key)
	err := setCmd.SAdd(key, &ObjectStruct{
		Name: "name",
		Age:  89,
	})
	if err != nil {
		t.Error(err)
	}
	var structs []*ObjectStruct
	fmt.Println(setCmd.SMembersScan(key, &structs))
	fmt.Println(json.ToJson(structs))

	fmt.Println(setCmd.SRem(key, []interface{}{&ObjectStruct{Name: "name", Age: 89}}))
	fmt.Println(setCmd.SMembersScan(key, &structs))

}
