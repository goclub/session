package sess

import (
	"context"
	"time"
)

type Session struct {
	sessionID string
	storeKey  string
	hub       Hub
	rw        SessionHttpReadWriter
}

func (s Session) existed(ctx context.Context) (existed bool, err error) {
	return s.hub.store.StoreKeyExists(ctx, s.storeKey)
}
func (s Session) ID() (sessionID string) {
	return s.sessionID
}
func (s Session) Get(ctx context.Context, field string) (value string, hasValue bool, err error) {
	return s.hub.store.Get(ctx, s.storeKey, field)
}
func (s Session) Set(ctx context.Context, field string, value string) (err error) {
	return s.hub.store.Set(ctx, s.storeKey, field, value)
}
func (s Session) Delete(ctx context.Context, field string) (err error) {
	return s.hub.store.Delete(ctx, s.storeKey, field)
}
func (s Session) Destroy(ctx context.Context) (err error) {
	// 如果是 cookie 场景则需要删除 cookie
	err = s.rw.Destroy(ctx, s.hub.option) // indivisible begin
	if err != nil {                       // indivisible end
		return
	}
	return s.hub.store.Destroy(ctx, s.storeKey)
}
func (s Session) SessionRemainingTTL(ctx context.Context) (ttl time.Duration, err error) {
	return s.hub.store.StoreKeyRemainingTTL(ctx, s.storeKey)
}
