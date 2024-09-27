package test

import (
	"fmt"
	"github.com/golang-acexy/starter-redis/redisstarter"
	"github.com/redis/go-redis/v9"
	"testing"
)

func TestSortedSet(t *testing.T) {
	cmd := redisstarter.SortedSetCmd()
	key := redisstarter.RedisKey{
		KeyFormat: "key-sorted-set",
	}
	cmd.ZAdds(key, []redis.Z{
		{
			Score:  1,
			Member: "member-1",
		},
		{
			Score:  2,
			Member: "member-2",
		},
	})

	fmt.Println(cmd.ZCount(key, 1, 1))
	fmt.Println(cmd.ZRange(key, 0, 0))
}
