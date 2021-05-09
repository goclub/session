package sess

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"net/http"
	"time"
)

// Hub 利用 Store 生成 sessionID 和查找 Session
type Hub struct {
	store Store
	option HubOption
}
func NewHub(store Store, option HubOption) (hub *Hub, err error) {
	// http cookie name 默认值
	if option.Cookie.Name == "" {
		option.Cookie.Name = "session_id"
	}
	// http header key 默认值
	if option.Header.Key == "" {
		option.Header.Key = "token"
	}
	// 默认加密解密方式
	if option.Security == nil {
		option.Security = DefaultSecurity{}
	}
	switch  option.Security.(type) {
	case DefaultSecurity:
		if len(option.SecureKey) != 32 {
			return nil, errors.New("goclub/sesison:  NewHub(store, option) option.SecureKey length must be 32")
		}
	}
	// 默认 sesison ttl
	if option.SessionTTL == 0 {
		option.SessionTTL = time.Hour * 2
	}
	if store == nil {
		return nil, errors.New("goclub/sesison: NewHub(store, option) store can not be nil")
	}

	hub = &Hub{
		store: store,
		option: option,
	}
	return hub, nil
}

type HubOption struct {
	// (必填) sessionID 与 storeKey 的加密解密秘钥 设置长度为 32 的 []byte
	SecureKey []byte
	// cookie 相关设置
	Cookie HubOptionCookie
	// sesison 过期时间，默认2小时
	SessionTTL time.Duration
	// header 相关设置
	Header HubOptionHeader
	// header 相关设置
	// 加密方式，不填则为 goclub/sesion 默认 aes 加密
	Security Security
	// 当sessionID 解码为 storeKey 后在 store 中不存在时触发
	// 用于监控系统排查恶意攻击或 sessionID 过期
	// ctx 可以用 ctx.WithValue 传递 requestID 便于排查问题
	OnStoreKeyDoesNotExist func(ctx context.Context, sessionID string, storeKey string)
	// 当请求的 sessionID 为空字符串时触发
	// 用于监控系统排查问题
	// ctx 可以用 ctx.WithValue 传递 requestID 便于排查问题
	OnRequestSessionIDIsEmptyString func(ctx context.Context)
}
type HubOptionCookie struct {
	// Name 建议设置为 项目名 + "_sesison_id" (若留空则为session_id)
	Name string
	Path   string
	Domain string
	MaxAge   int
	Secure   bool
}
type HubOptionHeader struct {
	// Key 建议设置为 token  (若留空则为 token)
	Key string
}


// 微信小程序和 app 场景下可能在登录成功时可能需要手动创建 SessionID
// 所以提供 NewSessionID 发放
func (hub Hub) NewSessionID(ctx context.Context) (sessionID string, err error) {
	storeKey :=  uuid.New().String()
	var sessionIDBytes []byte
	sessionIDBytes, err = hub.option.Security.Encrypt([]byte(storeKey), hub.option.SecureKey) ; if err != nil {
		return
	}
	sessionID =  string(sessionIDBytes)
	err = hub.store.InitSession(ctx, storeKey, hub.option.SessionTTL) ; if err != nil {
		return
	}
	return sessionID, nil
}

func (hub Hub) GetSessionBySessionID(ctx context.Context, sessionID string) (session Session, sessionExpired bool, err error) {
	session, has, err := hub.getSession(ctx, sessionID, nil) ; if err != nil {
	    return
	}
	// GetSessionBySessionID 的场景一般是微信小程序或 app
	// 这种场景当 key 错误应当返回key已过期 ，便于调用方告知客户端需要重新登录
	// 过期 = key 不存在
	sessionExpired = (has == false)
	return
}
func (hub Hub) getSession(ctx context.Context, sessionID string, writer http.ResponseWriter) (session Session, has bool, err error) {
	var storeKey string
	if sessionID == "" {
		// session 为空时候返回 has = false
		// 如果返回错误，会降低 goclub/session 的易用性
		if hub.option.OnRequestSessionIDIsEmptyString != nil {
			hub.option.OnRequestSessionIDIsEmptyString(ctx)
		}
		return Session{}, false, nil
	}
	var storeKeyBytes []byte
	storeKeyBytes, err = hub.option.Security.Decrypt([]byte(sessionID), hub.option.SecureKey) ; if err != nil {
		return Session{}, false,err
	}
	storeKey = string(storeKeyBytes)
	session = Session{
		sessionID: sessionID,
		storeKey: storeKey,
		hub: hub,
		writer: writer,
	}
	// 此处的验证可避免 key 过期或恶意猜测key进行攻击
	has, err = session.existed(ctx) ; if err != nil {
		return
	}
	if has == false && hub.option.OnStoreKeyDoesNotExist != nil {
		hub.option.OnStoreKeyDoesNotExist(ctx, sessionID, storeKey)
	}
	// 实现自动续期
	remainingTTL, err := session.hub.store.StoreKeyRemainingTTL(ctx, session.storeKey) ; if err != nil {
		return
	}
	if remainingTTL < session.hub.option.SessionTTL / 2 {
		err = session.hub.store.RenewTTL(ctx, session.storeKey, session.hub.option.SessionTTL) ; if err != nil {
			return
		}
	}
	return
}

func (hub Hub) GetSessionByCookie(ctx context.Context, writer http.ResponseWriter, request *http.Request) (Session, error) {
	rw := CookieReadWriter{
		Writer:  writer,
		Request: request,
	}
	s, err := hub.GetSessionByReadWriter(ctx, rw)
	return s, err
}
func (hub Hub) GetSessionByHeader(ctx context.Context, writer http.ResponseWriter, header http.Header) (Session, error) {
	rw := HeaderReadWriter{
		Writer: writer,
		Header: header,
	}
	s, err := hub.GetSessionByReadWriter(ctx, rw)
	return s, err
}
