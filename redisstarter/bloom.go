package redisstarter

import (
	"context"
	"errors"
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
func (*cmdBloom) Reserve(ctx context.Context, key RedisKey, errorRate float64, capacity int64, keyAppend ...interface{}) error {
	return redisClient.Do(ctx, "BF.RESERVE", OriginKeyString(key.KeyFormat, keyAppend...), errorRate, capacity).Err()
}

// Info 布隆过滤器信息
func (*cmdBloom) Info(ctx context.Context, key RedisKey, keyAppend ...interface{}) (*BloomInfo, error) {
	result, err := redisClient.Do(ctx, "BF.INFO", OriginKeyString(key.KeyFormat, keyAppend...)).Result()
	if err != nil {
		return nil, err
	}
	mapResult, ok := result.(map[interface{}]interface{})
	if !ok {
		return nil, errors.New("unknown redis response")
	}
	return &BloomInfo{
		Capacity:              mapResult["Capacity"].(int64),
		Size:                  mapResult["Size"].(int64),
		NumberOfFilters:       mapResult["Number of filters"].(int64),
		NumberOfItemsInserted: mapResult["Number of items inserted"].(int64),
		ExpansionRate:         mapResult["Expansion rate"].(int64),
	}, nil
}

// Add 向布隆过滤器中添加元素
func (*cmdBloom) Add(ctx context.Context, key RedisKey, value string, keyAppend ...interface{}) error {
	return redisClient.Do(ctx, "BF.ADD", OriginKeyString(key.KeyFormat, keyAppend...), value).Err()
}

// MAdd 向布隆过滤器中批量添加元素
func (*cmdBloom) MAdd(ctx context.Context, key RedisKey, values []string, keyAppend ...interface{}) error {
	args := make([]interface{}, 0)
	args = append(args, "BF.MADD")
	args = append(args, OriginKeyString(key.KeyFormat, keyAppend...))
	for _, v := range values {
		args = append(args, v)
	}
	return redisClient.Do(ctx, args...).Err()
}

// Exists 检查元素是否存在
func (*cmdBloom) Exists(ctx context.Context, key RedisKey, value string, keyAppend ...interface{}) (bool, error) {
	return redisClient.Do(ctx, "BF.EXISTS", OriginKeyString(key.KeyFormat, keyAppend...), value).Bool()
}

// MExists 检查多个元素是否存在
func (*cmdBloom) MExists(ctx context.Context, key RedisKey, values []string, keyAppend ...interface{}) ([]bool, error) {
	args := make([]interface{}, 0)
	args = append(args, "BF.MEXISTS")
	args = append(args, OriginKeyString(key.KeyFormat, keyAppend...))
	for _, v := range values {
		args = append(args, v)
	}
	return redisClient.Do(ctx, args...).BoolSlice()
}
