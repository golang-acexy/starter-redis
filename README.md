# starter-redis

`starter-redis` integrates `github.com/redis/go-redis/v9` with the Golang Acexy lifecycle. It manages one application-wide Redis universal client and provides typed key definitions, grouped command APIs, Pub/Sub, Bloom filter operations, and distributed locks.

## Ecosystem Role

Use this module when an application needs direct Redis capabilities. Higher-level cache policies such as local memory, cross-node invalidation, and L1/L2 caching belong to [`cloud-cache`](https://github.com/golang-acexy/cloud-cache), which uses this starter as its Redis foundation.

## Requirements

- Go `1.26.7`
- Redis standalone, Sentinel, or Cluster endpoints supported by `redis.UniversalOptions`
- `starter-parent` for coordinated lifecycle management

## Installation

```bash
go get github.com/golang-acexy/starter-redis
```

## Parent Lifecycle

```go
starter := &redisstarter.RedisStarter{
    Config: redisstarter.RedisConfig{
        UniversalOptions: redis.UniversalOptions{
            Addrs:    []string{"127.0.0.1:6379"},
            Password: "YOUR_PASSWORD",
            DB:       0,
        },
        InitFunc: func(client redis.UniversalClient) {
            // Register application-specific initialization here.
        },
    },
}

loader := parent.InitStarterLoader([]parent.Starter{starter})
if err := loader.Start(); err != nil {
    panic(err)
}
```

Use `LazyConfig` to resolve configuration on first use. It takes precedence over `Config` and may be evaluated during parent settings lookup before startup. Configuration is cached; configure the starter before use. `InitFunc` runs through the default parent settings, so custom `RedisSetting` must provide any required initialization callback itself.

The starter validates the connection with `PING`, creates the distributed-lock client, and exposes both clients only after successful startup. Stop Redis through the parent loader so Pub/Sub subscriptions and pooled connections are closed together.

## Redis Keys

Key-based grouped operations accept `RedisKey`; batch string commands also accept raw string keys:

```go
var UserSessionKey = redisstarter.NewRedisKey("user:session:%d", 30*time.Minute)

rawKey := UserSessionKey.RawKeyString(1001)
```

Key arguments are formatted with `fmt.Sprintf(KeyFormat, keyArgs...)`. Expiration is command-specific: string writes use it, hash/set writes atomically install initial expiration without refreshing existing TTL, and list/sorted-set/Bloom operations ignore it. Batch string writes do not set TTL. Lock acquisition requires a positive duration.

## Command Groups

Commands are grouped by Redis data structure:

- `StringCmd()` for string values.
- `HashCmd()` for hashes.
- `ListCmd()` for lists.
- `SetCmd()` for sets.
- `SortedSetCmd()` for sorted sets.
- `KeyCmd()` for key-level operations.
- `TopicCmd()` for Pub/Sub.
- `BloomCmd()` for RedisBloom operations.

Example:

```go
key := redisstarter.NewRedisKey("profile:%d", time.Hour)

if err := redisstarter.StringCmd().Set(key, profileJSON, userID); err != nil {
    return err
}

value, err := redisstarter.StringCmd().Get(key, userID)
```

## Pub/Sub

`TopicCmd` manages subscriptions using the shared Redis client. Define stable topic keys and ensure the starter remains active for the complete subscription lifetime. Parent-managed shutdown closes registered subscriptions before closing the client. `SubscribeRepeat` gives each caller a separate channel and cancellation function. Slow consumers can lose events; received message pointers are shared and should be treated as read-only. Retry subscriptions do not replay missed messages.

## Distributed Locks

The starter wraps `bsm/redislock` and provides callback and explicit-lock styles:

```go
lockKey := redisstarter.NewRedisKey("lock:order:%d", 10*time.Second)

ctx := context.Background()
err := redisstarter.TryLockWithContext(ctx, lockKey, func() error {
    // Execute the protected operation and return its error.
    return nil
}, orderID)
if err != nil {
    return err
}
```

The callback executes synchronously. The returned error includes callback and unlock failures. Use `ObtainLockerWithContext` when the caller must control unlock timing or refresh a long-running lock. Advanced lock token and metadata settings are available through the corresponding `WithOptions` APIs. `LockWithMaxRetry` and `LockWithDeadline` support bounded retry policies. Locks do not automatically renew or stop callback execution on context cancellation. Explicitly refresh long-running leases and account for ownership loss before business work completes. Release uses a separate bounded context so acquisition cancellation does not prevent cleanup.

## Raw Access

```go
client := redisstarter.RawRedisClient()
locker := redisstarter.RawLockerClient()
```

Raw access is intended for Redis commands not covered by the grouped APIs. The starter must already be running, and callers must not close the shared clients.

## Common Errors

Reusable lifecycle and lock errors are declared in `redisstarter/error.go`. In particular, operations require a started Redis client, duplicate startup is rejected, and lock acquisition can fail because of contention, cancellation, or invalid configuration.

## Usage Constraints

- One process owns one Redis universal client and one lock client.
- The starter accepts the same endpoint models as `redis.UniversalOptions`, including standalone, Sentinel, and Cluster configurations.
- Most grouped data commands use background contexts. Use raw contextual commands when deadlines or cancellation are required. Older helpers can panic before startup; scalar results may hide command errors.
- Positive-expiry hash/set writes require Lua execution plus TTL and EXPIRE permissions. Script errors are returned; Redis does not roll back writes already completed within a failing script.
- The standard Redis starter does not allow parent-managed restart after successful shutdown.
- RedisBloom commands require BF.* server support; Bloom membership can have false positives. `BloomCmd().Info` supports both RESP2 array replies and RESP3 map replies.

## Testing

Default tests are deterministic package-level unit tests and do not require Redis:

```bash
go test ./...
```

Tests under `test/` exercise a real standalone or cluster deployment and are guarded by the `integration` build tag. Configure their Redis endpoints for the target environment before running them explicitly:

```bash
go test -tags=integration ./test
```
