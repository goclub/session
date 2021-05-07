package sess

import (
	"context"
	"errors"
	"github.com/go-redis/redis/v8"
	"strconv"
	"time"
)

func NewRedisStore(option RedisStoreOption) RedisStore {
	return RedisStore{
		option: option,
	}
}
type RedisStoreOption struct {
	Client redis.UniversalClient
	StoreKeyPrefix string
}
type RedisStore struct {
	option RedisStoreOption
}
func (m RedisStore) getKey(storeKey string) (key string){
	return m.option.StoreKeyPrefix + ":"+ storeKey
}
func (m RedisStore) InitSession(ctx context.Context, storeKey string, sessionTTL time.Duration) (err error) {
	key := m.getKey(storeKey)
	client := m.option.Client
	// hset key __goclub_session_create_time time.Now() 是为了让 key 存在
	script := `
	local key = KEYS[1]
	local field = KEYS[2]
	local nowUnix = ARGV[1]
	local ttl = ARGV[2]
	redis.call("hset", key, field, nowUnix)
	return redis.call("pexpire", key, ttl)
	`
	field := "__goclub_session_create_time"
	evalKeys := []string{key, field}
	argv := []interface{}{strconv.FormatInt(time.Now().Unix(), 10) , sessionTTL.Milliseconds()}
	result, err := client.Eval(ctx, script, evalKeys, argv...).Result() ; if err != nil {
		return
	}
	intReply := result.(int64)
	if intReply == 0 {
		// 理论上 pexpire 不会返回 0 ，但是严谨一点应当在返回 0 时候返回错误
		return errors.New("goclub/session: RedisStore NewSession redis pexpire fail, key is " + key)
	}
	return
}
func (m RedisStore) StoreKeyExists(ctx context.Context, storeKey string) (existed bool, err error) {
	key := m.getKey(storeKey)
	client := m.option.Client
	reply, err := client.Exists(ctx, key).Result() ; if err != nil {
		return
	}
	existed = reply == 1
	return
}
func (m RedisStore) StoreKeyRemainingTTL(ctx context.Context, storeKey string) (remainingTTL time.Duration, err error) {
	key := m.getKey(storeKey)
	client := m.option.Client
	return client.PTTL(ctx, key).Result()
}
func (m RedisStore) RenewTTL(ctx context.Context, storeKey string, ttl time.Duration) (err error) {
	key := m.getKey(storeKey)
	client := m.option.Client
	return client.PExpire(ctx, key, ttl).Err()
}
func (m RedisStore) Get(ctx context.Context, storeKey string, field string) (value string, hasValue bool, err error) {
	key := m.getKey(storeKey)
	client := m.option.Client
	hasValue = true
	cmd := client.HGet(ctx, key, field)
	value, err = cmd.Result() ; if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", false, nil
		} else {
			return
		}
	}
	return
}
func (m RedisStore) Set(ctx context.Context, storeKey string, field string, value string) (err error) {
	key := m.getKey(storeKey)
	client := m.option.Client
	cmd := client.HSet(ctx, key, field, value)
	_, err = cmd.Result() ; if err != nil {
		return
	}
	return
}
func (m RedisStore) Delete(ctx context.Context, storeKey string, field string) (err error) {
	key := m.getKey(storeKey)
	client := m.option.Client
	cmd := client.HDel(ctx, key, field)
	_, err = cmd.Result() ; if err != nil {
		return
	}
	return
}

func (m RedisStore) Destroy(ctx context.Context,storeKey string) (err error){
	key := m.getKey(storeKey)
	client := m.option.Client
	cmd := client.Del(ctx, key)
	_, err = cmd.Result() ; if err != nil {
		return
	}
	return
}

