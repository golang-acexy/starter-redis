package redisstarter

import (
	"context"
	"math"
	"strconv"
)

type cmdBloom struct {
}

var bloomCmd = new(cmdBloom)

func BloomCmd() *cmdBloom {
	return bloomCmd
}

type BloomInfo struct {
	// 初始容量
	Capacity int64
	// 底层大小bit
	Size int64
	// 分片数量
	NumberOfFilters int64
	// 已插入总元素
	NumberOfItemsInserted int64
	// 拓展速率
	ExpansionRate int64
}

// Reserve 创建布隆过滤器
func (*cmdBloom) Reserve(key RedisKey, errorRate float64, capacity int64, keyAppend ...any) error {
	return redisClient.Do(context.Background(), "BF.RESERVE", key.RawKeyString(keyAppend...), errorRate, capacity).Err()
}

// Info 布隆过滤器信息
func (*cmdBloom) Info(key RedisKey, keyAppend ...any) (*BloomInfo, error) {
	result, err := redisClient.Do(context.Background(), "BF.INFO", key.RawKeyString(keyAppend...)).Result()
	if err != nil {
		return nil, err
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

// Add 向布隆过滤器中添加元素
func (*cmdBloom) Add(key RedisKey, value string, keyAppend ...any) error {
	return redisClient.Do(context.Background(), "BF.ADD", key.RawKeyString(keyAppend...), value).Err()
}

// MAdd 向布隆过滤器中批量添加元素
func (*cmdBloom) MAdd(key RedisKey, values []string, keyAppend ...any) error {
	args := make([]any, 0)
	args = append(args, "BF.MADD")
	args = append(args, key.RawKeyString(keyAppend...))
	for _, v := range values {
		args = append(args, v)
	}
	return redisClient.Do(context.Background(), args...).Err()
}

// Exists 检查元素是否存在
func (*cmdBloom) Exists(key RedisKey, value string, keyAppend ...any) (bool, error) {
	return redisClient.Do(context.Background(), "BF.EXISTS", key.RawKeyString(keyAppend...), value).Bool()
}

// MExists 检查多个元素是否存在
func (*cmdBloom) MExists(key RedisKey, values []string, keyAppend ...any) ([]bool, error) {
	args := make([]any, 0)
	args = append(args, "BF.MEXISTS")
	args = append(args, key.RawKeyString(keyAppend...))
	for _, v := range values {
		args = append(args, v)
	}
	return redisClient.Do(context.Background(), args...).BoolSlice()
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
