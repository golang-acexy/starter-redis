package redisstarter

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Run the write and initial expiration without allowing other commands to interleave.
// Redis script errors are returned, but Redis does not roll back preceding script writes.
var writeWithInitialExpiryScript = redis.NewScript(`
local result = redis.call(ARGV[1], KEYS[1], unpack(ARGV, 3))
if redis.call('TTL', KEYS[1]) == -1 then
    redis.call('EXPIRE', KEYS[1], ARGV[2])
end
return result
`)

func writeWithInitialExpiry(command, key string, expiry time.Duration, values []any) error {
	// Preserve the previous go-redis Expire precision and minimum positive duration.
	seconds := int64(expiry / time.Second)
	if seconds < 1 {
		seconds = 1
	}
	args := make([]any, 0, len(values)+2)
	args = append(args, command, seconds)
	args = append(args, values...)
	return writeWithInitialExpiryScript.Run(context.Background(), rawRedisClient(), []string{key}, args...).Err()
}
