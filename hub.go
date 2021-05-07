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
func NewHub(store Store, option HubOption) *Hub {
	if option.Security == nil {
		option.Security = DefaultSecurity{}
	}
	return &Hub{
		store: store,
		option: option,
	}
}

type HubOption struct {
	// sessionID 与 storeKey 的加密解密秘钥
	// 设置长度为 32的 []byte
	SecureKey []byte
	// cookie 相关设置
	Cookie HubOptionCookie
	// 加密方式，不填则为 goclub/sesion 默认 aes 加密
	Security Security
	// sesison 过期时间，建议为2小时
	SessionTTL time.Duration
	// 当sessionID 解码为 storeKey 后在 store 中不存在时触发
	// 可结合监控系统排查恶意攻击或 sessionID 过期
	OnStoreKeyDoesNotExist func(sessionID string, storeKey string)
}
type HubOptionCookie struct {
	// Name 建议设置为 项目名 + "_sesison_id"
	Name string
	Path   string
	Domain string
	MaxAge   int
	Secure   bool
}



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
		return Session{}, false,&ErrEmptySessionID{}
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
		hub.option.OnStoreKeyDoesNotExist(sessionID, storeKey)
	}
	return
}




type SessionIDReadWriter interface {
	Read(ctx context.Context, hubOption HubOption) (sessionID string, has bool, err error)
	Write(ctx context.Context, hubOption HubOption, sessionID string) (err error)
}
func (hub Hub) GetSessionByReadWriter( ctx context.Context, rw SessionIDReadWriter) (session Session, err error) {
	sessionID, has, err := rw.Read(ctx, hub.option) ; if err != nil {
	    return
	}
	// 如果客户端没有session 则生成新的 session
	if has == false {
		sessionID, err = hub.NewSessionID(ctx) ; ; if err != nil {
		    return
		}
	}
	session, hasSession, err := hub.getSession(ctx, sessionID, nil) ; if err != nil {
	    return
	}
	// session 如果过期和恶意攻击的情况 会 hasSession == false
	// (可以在已经 NewSessionID 之后清除 store 的数据以测试这种情况,例如 redis flushdb)
	if hasSession == false {
		// 这种两种情况都生成新的 session
		sessionID, err :=  hub.NewSessionID(ctx) ; if err != nil {
			return Session{},err
		}
		// 生成新的 sessionID 后再返回 Session{}
		session, _, err = hub.getSession(ctx, sessionID, nil) ; if err != nil {
			return Session{}, err
		}
		// 同时更新客户端 sessionID
		err = rw.Write(ctx, hub.option, sessionID) ; if err != nil {
		    return Session{}, err
		}
	}
	return session, nil
}

type CookieReadWriter struct {
	Writer http.ResponseWriter
	Request *http.Request
}
func (rw CookieReadWriter) Read(ctx context.Context, hubOption HubOption) (sessionID string, has bool, err error) {
	var noCookie bool
	cookie, err := rw.Request.Cookie(hubOption.Cookie.Name) ; if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			noCookie = true
		} else {
			return "", false,err
		}
	}
	if noCookie {
		return "", false, nil
	}
	return cookie.Value, true, nil
}
func (rw CookieReadWriter) Write(ctx context.Context,hubOption HubOption,  sessionID string) (err error) {
	http.SetCookie(rw.Writer, &http.Cookie{
		Name:       hubOption.Cookie.Name,
		Value:      sessionID,
		Path:       hubOption.Cookie.Path,
		Domain:     hubOption.Cookie.Domain,
		MaxAge:     hubOption.Cookie.MaxAge,
		Secure:     hubOption.Cookie.Secure,
		HttpOnly:   true,
	})
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
