package sess

import (
	"context"
	xerr "github.com/goclub/error"
	red "github.com/goclub/redis"
	"strconv"
	"time"
)

func NewRedisStore(option RedisStoreOption) RedisStore {
	return RedisStore{
		option: option,
	}
}
type RedisStoreOption struct {
	Client red.Connecter
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
	// lua 保证原子性
	// hset key __goclub_session_create_time time.Now() 是为了让 key 存在
	script := `
	local key = KEYS[1]
	local field = KEYS[2]
	local nowUnix = ARGV[1]
	local ttl = ARGV[2]
	redis.call("HSET", key, field, nowUnix)
	return redis.call("pexpire", key, ttl)
	`
	field := "__goclub_session_create_time"
	evalKeys := []string{key, field}
	argv := []string{strconv.FormatInt(time.Now().Unix(), 10) , strconv.FormatInt(sessionTTL.Milliseconds(), 10)}
	result, isNil,  err := client.Eval(ctx, red.Script{
		Keys:   evalKeys,
		Argv:   argv,
		Script: script,
	}) ; if err != nil {
		return
	}
	if isNil {
		return xerr.New("goclub/session: RedisStore InitSession redis can not be nil")
	}
	intReply := result.(int64)
	if intReply == 0 {
		// 理论上 pexpire 不会返回 0 ，但是严谨一点应当在返回 0 时候返回错误
		return xerr.New("goclub/session: RedisStore NewSession redis pexpire fail, key is " + key)
	}
	return
}
func (m RedisStore) StoreKeyExists(ctx context.Context, storeKey string) (existed bool, err error) {
	key := m.getKey(storeKey)
	client := m.option.Client
	reply, err := red.EXISTS{
		Key:  key,
	}.Do(ctx, client) ; if err != nil {
		return
	}
	existed = reply == 1
	return
}
func (m RedisStore) StoreKeyRemainingTTL(ctx context.Context, storeKey string) (remainingTTL time.Duration, err error) {
	key := m.getKey(storeKey)
	client := m.option.Client
	result, err := red.PTTL{
		Key:  key,
	}.Do(ctx, client) ; if err != nil {
	    return
	}
	return result.TTL, nil
}
func (m RedisStore) RenewTTL(ctx context.Context, storeKey string, ttl time.Duration) (err error) {
	key := m.getKey(storeKey)
	client := m.option.Client
	_, err = red.PEXPIRE{
		Key: key,
		Duration: ttl,
	}.Do(ctx, client) ; if err != nil {
	    return
	}
	return
}
func (m RedisStore) Get(ctx context.Context, storeKey string, field string) (value string, hasValue bool, err error) {
	key := m.getKey(storeKey)
	client := m.option.Client
	hasValue = true
	value, isNil, err := client.DoStringReply(ctx, []string{"HGET", key, field}) ; if err != nil {
	    return
	}
	if isNil {
		return "", false, nil
	}
	return
}
func (m RedisStore) Set(ctx context.Context, storeKey string, field string, value string) (err error) {
	key := m.getKey(storeKey)
	client := m.option.Client
	_, err = client.DoIntegerReplyWithoutNil(ctx, []string{"HSET", key, field, value}) ; if err != nil {
		return
	}
	return
}
func (m RedisStore) Delete(ctx context.Context, storeKey string, field string) (err error) {
	key := m.getKey(storeKey)
	client := m.option.Client
	_, err = client.DoIntegerReplyWithoutNil(ctx, []string{"HDEL", key, field}) ; if err != nil {
		return
	}
	return
}

func (m RedisStore) Destroy(ctx context.Context,storeKey string) (err error){
	key := m.getKey(storeKey)
	client := m.option.Client
	_, err = red.DEL{Key: key}.Do(ctx, client) ; if err != nil {
	    return
	}
	return
}
