package sess

import (
	"context"
	"net/http"
	"time"
)

type Session struct {
	sessionID string
	storeKey string
	hub Hub
	writer http.ResponseWriter
}

func (s Session) existed(ctx context.Context) (existed bool, err error) {
	return s.hub.store.StoreKeyExists(ctx, s.storeKey)
}
func (s Session) ID() (sessionID string) {
	return s.sessionID
}
func (s Session) Get(ctx context.Context, field string) (value string, hasValue bool, err error) {
	// 只需要 get 时检查续期，因为 get delete 之前必定有 get
	remainingTTL, err := s.hub.store.StoreKeyRemainingTTL(ctx, s.storeKey) ; if err != nil {
	    return
	}
	if remainingTTL < s.hub.option.SessionTTL / 2 {
		err = s.hub.store.RenewTTL(ctx, s.storeKey, s.hub.option.SessionTTL) ; if err != nil {
		    return
		}
	}
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
	if s.writer != nil {
		cookie := http.Cookie{
			Name: s.hub.option.Cookie.Name,
			Value: "",
			Path: s.hub.option.Cookie.Path,
			Domain: s.hub.option.Cookie.Domain,
			Secure: s.hub.option.Cookie.Secure,
			MaxAge: -1, // -1 标识清除cookie
		}
		http.SetCookie(s.writer, &cookie)
	}
	return s.hub.store.Destroy(ctx, s.storeKey)
}
func (s Session) SessionRemainingTTL(ctx context.Context) (ttl time.Duration, err error) {
	return s.hub.store.StoreKeyRemainingTTL(ctx, s.storeKey)
}
