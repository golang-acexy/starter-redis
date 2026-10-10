package redisstarter

import (
	"context"
	"errors"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

type ttlResultHook struct {
	ttl time.Duration
	err error
}

// DialHook leaves transport dialing unchanged; the command hook bypasses it in these tests.
func (h ttlResultHook) DialHook(next redis.DialHook) redis.DialHook { return next }

// ProcessPipelineHook leaves pipeline behavior unchanged.
func (h ttlResultHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}

// ProcessHook supplies driver duration results without requiring a Redis server.
func (h ttlResultHook) ProcessHook(_ redis.ProcessHook) redis.ProcessHook {
	return func(_ context.Context, cmd redis.Cmder) error {
		cmd.(*redis.DurationCmd).SetVal(h.ttl)
		cmd.SetErr(h.err)
		return h.err
	}
}

func useRegressionClient(t *testing.T, client redis.UniversalClient) {
	t.Helper()
	previous := redisRuntimeState.Swap(&redisRuntime{client: client})
	t.Cleanup(func() { redisRuntimeState.Store(previous) })
}

func TestKeyTtlPreservesProtocolSentinels(t *testing.T) {
	commandErr := errors.New("TTL command failed")
	for _, test := range []struct {
		name string
		ttl  time.Duration
		err  error
		want float64
	}{
		{name: "remaining seconds", ttl: 12 * time.Second, want: 12},
		{name: "final second", ttl: 0, want: 0},
		{name: "persistent", ttl: -1, want: -1},
		{name: "missing", ttl: -2, want: -2},
		{name: "command error", err: commandErr, want: -3},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := redis.NewClient(&redis.Options{})
			t.Cleanup(func() { _ = client.Close() })
			client.AddHook(ttlResultHook{ttl: test.ttl, err: test.err})
			useRegressionClient(t, client)
			if got := KeyCmd().Ttl(NewRedisKey("ttl:%d"), 1); got != test.want {
				t.Fatalf("TTL = %v, want %v", got, test.want)
			}
		})
	}
}

func TestBloomInfoAcceptsBothProtocols(t *testing.T) {
	want := &BloomInfo{Capacity: 100, Size: 240, NumberOfFilters: 1, NumberOfItemsInserted: 8, ExpansionRate: 2}
	for _, reply := range []any{
		[]any{"Capacity", int64(100), "Size", int64(240), "Number of filters", int64(1), "Number of items inserted", int64(8), "Expansion rate", int64(2)},
		map[any]any{"Capacity": int64(100), "Size": int64(240), "Number of filters": int64(1), "Number of items inserted": int64(8), "Expansion rate": int64(2)},
	} {
		got, err := parseBloomInfo(reply)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("reply %T: info = %+v, err = %v", reply, got, err)
		}
	}
}

func TestBloomInfoRejectsMalformedReplies(t *testing.T) {
	for _, reply := range []any{
		nil,
		[]any{},
		[]any{"Capacity"},
		[]any{int64(1), int64(100)},
		[]any{"Capacity", int64(100), "Capacity", int64(100)},
		map[any]any{"Capacity": int64(100)},
		map[any]any{"Capacity": uint64(math.MaxUint64), "Size": int64(240), "Number of filters": int64(1), "Number of items inserted": int64(8), "Expansion rate": int64(2)},
		map[any]any{"Capacity": "invalid", "Size": int64(240), "Number of filters": int64(1), "Number of items inserted": int64(8), "Expansion rate": int64(2)},
	} {
		got, err := parseBloomInfo(reply)
		if got != nil || !errors.Is(err, ErrUnknownRedisResponse) {
			t.Fatalf("reply %#v: info = %+v, err = %v", reply, got, err)
		}
	}
}
