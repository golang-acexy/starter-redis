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
func (*cmdBloom) Reserve(key RedisKey, errorRate float64, capacity int64, keyAppend ...interface{}) error {
	return redisClient.Do(context.Background(), "BF.RESERVE", key.RawKeyString(keyAppend...), errorRate, capacity).Err()
}

// Info 布隆过滤器信息
func (*cmdBloom) Info(key RedisKey, keyAppend ...interface{}) (*BloomInfo, error) {
	result, err := redisClient.Do(context.Background(), "BF.INFO", key.RawKeyString(keyAppend...)).Result()
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
func (*cmdBloom) Add(key RedisKey, value string, keyAppend ...interface{}) error {
	return redisClient.Do(context.Background(), "BF.ADD", key.RawKeyString(keyAppend...), value).Err()
}

// MAdd 向布隆过滤器中批量添加元素
func (*cmdBloom) MAdd(key RedisKey, values []string, keyAppend ...interface{}) error {
	args := make([]interface{}, 0)
	args = append(args, "BF.MADD")
	args = append(args, key.RawKeyString(keyAppend...))
	for _, v := range values {
		args = append(args, v)
	}
	return redisClient.Do(context.Background(), args...).Err()
}

// Exists 检查元素是否存在
func (*cmdBloom) Exists(key RedisKey, value string, keyAppend ...interface{}) (bool, error) {
	return redisClient.Do(context.Background(), "BF.EXISTS", key.RawKeyString(keyAppend...), value).Bool()
}

// MExists 检查多个元素是否存在
func (*cmdBloom) MExists(key RedisKey, values []string, keyAppend ...interface{}) ([]bool, error) {
	args := make([]interface{}, 0)
	args = append(args, "BF.MEXISTS")
	args = append(args, key.RawKeyString(keyAppend...))
	for _, v := range values {
		args = append(args, v)
	}
	return redisClient.Do(context.Background(), args...).BoolSlice()
}
