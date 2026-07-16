# starter-redis

`starter-redis` integrates `github.com/redis/go-redis/v9` with the Golang Acexy lifecycle. It manages one application-wide Redis universal client and provides typed key definitions, grouped command APIs, Pub/Sub, Bloom filter operations, and distributed locks.

## Ecosystem Role

Use this module when an application needs direct Redis capabilities. Higher-level cache policies such as local memory, cross-node invalidation, and L1/L2 caching belong to [`cloud-cache`](https://github.com/golang-acexy/cloud-cache), which uses this starter as its Redis foundation.

## Requirements

- Go `1.25.8`
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

Use `LazyConfig` when configuration must be resolved immediately before startup. It takes precedence over `Config`.

The starter validates the connection with `PING`, creates the distributed-lock client, and exposes both clients only after successful startup. Stop Redis through the parent loader so Pub/Sub subscriptions and pooled connections are closed together.

## Redis Keys

All grouped command APIs accept `RedisKey`, keeping key formatting and expiration policy close to the key definition:

```go
var UserSessionKey = redisstarter.NewRedisKey("user:session:%d", 30*time.Minute)

rawKey := UserSessionKey.RawKeyString(1001)
```

Key arguments are formatted with `fmt.Sprintf(KeyFormat, keyArgs...)`. A zero expiration means no automatic expiration for normal data commands; lock APIs require a valid lock duration where documented.

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
ctx := context.Background()
key := redisstarter.NewRedisKey("profile:%d", time.Hour)

if err := redisstarter.StringCmd().Set(ctx, key, profileJSON, userID); err != nil {
    return err
}

value, err := redisstarter.StringCmd().Get(ctx, key, userID)
```

## Pub/Sub

`TopicCmd` manages subscriptions using the shared Redis client. Define stable topic keys and ensure the starter remains active for the complete subscription lifetime. Parent-managed shutdown closes registered subscriptions before closing the client.

## Distributed Locks

The starter wraps `bsm/redislock` and provides callback and explicit-lock styles:

```go
lockKey := redisstarter.NewRedisKey("lock:order:%d", 10*time.Second)

err, done := redisstarter.TryLockWithContext(ctx, lockKey, func() {
    // Execute the protected operation.
}, nil, orderID)
if err != nil {
    return err
}
<-done
```

Use `TryAndGetLockerWithContext` when the caller must control unlock timing. `LockWithMaxRetry` and `LockWithDeadline` support bounded retry policies. Always choose an expiration longer than the expected protected operation and propagate cancellation through context-aware APIs.

## Raw Access

```go
client := redisstarter.RawRedisClient()
locker := redisstarter.RawLockerClient()
```

Raw access is intended for Redis commands not covered by the grouped APIs. The starter must already be running, and callers must not close the shared clients.

## Common Errors

Reusable lifecycle and lock errors are declared in `redisstarter/error.go`. In particular, operations require a started Redis client, duplicate startup is rejected, and lock acquisition can fail because of contention, cancellation, or invalid configuration.

## Design Notes

- One process owns one Redis universal client and one lock client.
- The starter accepts the same endpoint models as `redis.UniversalOptions`, including standalone, Sentinel, and Cluster configurations.
- Standard commands use explicit contexts and centralized `RedisKey` definitions.
- The standard Redis starter does not allow parent-managed restart after successful shutdown.
- Use `cloud-cache` instead of building application cache synchronization directly on Pub/Sub.
