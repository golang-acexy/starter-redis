# starter-redis

基于`github.com/redis/go-redis`封装的集中式缓存组件

---

#### 功能说明

规范了Redis操作方法调用风格，统一RedisKey类型，统一过期时间设置，避免了全局写rediskey的问题，所有Redis操作方法只允许使用RedisKey类型作为key参数，方便集中定义，**支持布隆过滤器操作**

```go
type RedisKey struct {

	// 最终key值的格式化格式 将使用 fmt.Sprintf(key.KeyFormat, keyAppend) 进行处理
	KeyFormat string
	Expire    time.Duration
}
```

> 提供常用的操作方法并分类`cmdhash` `cmdkey` `cmdset` `cmdstring` `cmdothers`

```go
func TestSet(t *testing.T) {
	stringType := redisstarter.StringCmd()

	key1 := redisstarter.RedisKey{
		KeyFormat: "string:%d:%s",
		Expire:    time.Second,
	}
	_ = stringType.Set(context.Background(), key1, "你好", 1, "2")
	fmt.Println(stringType.Get(context.Background(), key1, 1, "2"))
	time.Sleep(time.Second * 2)
	fmt.Println(stringType.Get(context.Background(), key1, 1, "2"))
}

func TestSAdd(t *testing.T) {
	key := redisstarter.RedisKey{
		KeyFormat: "key-set",
	}
	err := setCmd.SAdd(context.Background(), key, []string{"你", "好"})
	if err != nil {
		t.Error(err)
	}
	fmt.Println(setCmd.SMembers(context.Background(), key))
}
```

> 依赖Redis简单的分布式锁操作

```go
func TestTryLock(t *testing.T) {
	number := 0
	key := random.RandString(5)
	for i := 0; i < 100; i++ {
		go tryLock(key, &number)
	}
	time.Sleep(time.Second * 5)
	fmt.Println(number)
}

func tryLock(k string, i *int) {
	err := redisstarter.TryLock(k, time.Minute, func() {
		*i = *i + 1
		fmt.Println(*i)
	})
	if err != nil {
		fmt.Printf("%+v %s \n", err, k)
		return
	}
}

func lock(ctx context.Context, key string, i *int) {
	err := redisstarter.LockWithDeadline(ctx, key, time.Minute, time.Now().Add(time.Minute), 200, func() {
		*i = *i + 1
		time.Sleep(time.Duration(random.RandRangeInt(100, 300)) * time.Millisecond)
		fmt.Println(*i)
	})
	if err != nil {
		fmt.Printf("%+v %s \n", err, key)
		return
	}
}
func TestLockWithDeadline(t *testing.T) {
	ctx := context.Background()
	deadline, cancel := context.WithDeadline(ctx, time.Now().Add(10*time.Second))
	number := 0
	key := random.RandString(5)
	for i := 0; i < 100; i++ {
		go lock(deadline, key, &number)
	}
	time.Sleep(time.Second * 5)
	cancel()
	fmt.Println(number)
}
```
