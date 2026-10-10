package redisstarter

import (
	"context"
	"math"
	"strconv"
)

type cmdBloom struct {
}

var bloomCmd = new(cmdBloom)

// BloomCmd returns shared Bloom helpers requiring RedisBloom BF.* support and a running starter. Calls
// can panic before startup, use background contexts, and do not apply key.Expire.
func BloomCmd() *cmdBloom {
	return bloomCmd
}

// BloomInfo contains server-reported filter statistics; item counts are probabilistic, not exact
// business cardinality.
type BloomInfo struct {
	// Capacity is the total reported item capacity across sub-filters.
	Capacity int64
	// Size is allocated memory in bytes, not bits.
	Size int64
	// NumberOfFilters is the number of sub-filters.
	NumberOfFilters int64
	// NumberOfItemsInserted counts additions detected as new, not exact distinct business items.
	NumberOfItemsInserted int64
	// ExpansionRate is the configured scaling factor.
	ExpansionRate int64
}

// Reserve creates a Bloom filter; errorRate and capacity are forwarded for server validation. An
// existing key can fail; no expiration is installed.
func (*cmdBloom) Reserve(key RedisKey, errorRate float64, capacity int64, keyAppend ...any) error {
	return rawRedisClient().Do(context.Background(), "BF.RESERVE", key.RawKeyString(keyAppend...), errorRate, capacity).Err()
}

// Info reads Bloom statistics from RESP2 arrays or RESP3 maps. Malformed replies or invalid field
// values return ErrUnknownRedisResponse; command errors are preserved.
func (*cmdBloom) Info(key RedisKey, keyAppend ...any) (*BloomInfo, error) {
	result, err := rawRedisClient().Do(context.Background(), "BF.INFO", key.RawKeyString(keyAppend...)).Result()
	if err != nil {
		return nil, err
	}
	return parseBloomInfo(result)
}

// parseBloomInfo normalizes RESP2 field/value pairs before validating the required statistics.
func parseBloomInfo(result any) (*BloomInfo, error) {
	if values, ok := result.([]any); ok {
		if len(values)%2 != 0 {
			return nil, ErrUnknownRedisResponse
		}
		fields := make(map[any]any, len(values)/2)
		for index := 0; index < len(values); index += 2 {
			name, ok := values[index].(string)
			if !ok {
				return nil, ErrUnknownRedisResponse
			}
			if _, exists := fields[name]; exists {
				return nil, ErrUnknownRedisResponse
			}
			fields[name] = values[index+1]
		}
		result = fields
	}
	mapResult, ok := result.(map[any]any)

	if !ok {
		return nil, ErrUnknownRedisResponse
	}
	capacity, ok := parseRedisInt64(mapResult["Capacity"])
	if !ok {
		return nil, ErrUnknownRedisResponse
	}
	size, ok := parseRedisInt64(mapResult["Size"])
	if !ok {
		return nil, ErrUnknownRedisResponse
	}
	numberOfFilters, ok := parseRedisInt64(mapResult["Number of filters"])
	if !ok {
		return nil, ErrUnknownRedisResponse
	}
	numberOfItemsInserted, ok := parseRedisInt64(mapResult["Number of items inserted"])
	if !ok {
		return nil, ErrUnknownRedisResponse
	}
	expansionRate, ok := parseRedisInt64(mapResult["Expansion rate"])
	if !ok {
		return nil, ErrUnknownRedisResponse
	}
	return &BloomInfo{
		Capacity:              capacity,
		Size:                  size,
		NumberOfFilters:       numberOfFilters,
		NumberOfItemsInserted: numberOfItemsInserted,
		ExpansionRate:         expansionRate,
	}, nil
}

// Add adds one value; duplicate or probably-present status is discarded, and only command errors are
// returned.
func (*cmdBloom) Add(key RedisKey, value string, keyAppend ...any) error {
	return rawRedisClient().Do(context.Background(), "BF.ADD", key.RawKeyString(keyAppend...), value).Err()
}

// MAdd adds values in one command; per-item insertion status is discarded. Empty input is passed to
// Redis and may cause a command error.
func (*cmdBloom) MAdd(key RedisKey, values []string, keyAppend ...any) error {
	args := make([]any, 0)
	args = append(args, "BF.MADD")
	args = append(args, key.RawKeyString(keyAppend...))
	for _, v := range values {
		args = append(args, v)
	}
	return rawRedisClient().Do(context.Background(), args...).Err()
}

// Exists reports probabilistic membership with possible false positives; it is not proof of business
// existence.
func (*cmdBloom) Exists(key RedisKey, value string, keyAppend ...any) (bool, error) {
	return rawRedisClient().Do(context.Background(), "BF.EXISTS", key.RawKeyString(keyAppend...), value).Bool()
}

// MExists reports probabilistic membership in input order; empty input is passed to Redis and may cause
// a command error.
func (*cmdBloom) MExists(key RedisKey, values []string, keyAppend ...any) ([]bool, error) {
	args := make([]any, 0)
	args = append(args, "BF.MEXISTS")
	args = append(args, key.RawKeyString(keyAppend...))
	for _, v := range values {
		args = append(args, v)
	}
	return rawRedisClient().Do(context.Background(), args...).BoolSlice()
}

func parseRedisInt64(value any) (int64, bool) {
	switch v := value.(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	case int32:
		return int64(v), true
	case uint64:
		if v > math.MaxInt64 {
			return 0, false
		}
		return int64(v), true
	case string:
		i, err := strconv.ParseInt(v, 10, 64)
		return i, err == nil
	case []byte:
		i, err := strconv.ParseInt(string(v), 10, 64)
		return i, err == nil
	default:
		return 0, false
	}
}
