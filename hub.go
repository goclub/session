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
	SecurityKey []byte
	Cookie HubOptionCookie
	Security Security
	SessionTTL time.Duration
}
type HubOptionCookie struct {
	Name string
	Path   string
	Domain string
	MaxAge   int
	Secure   bool
}



func (hub Hub) NewSessionID(ctx context.Context) (sessionID string, err error) {
	storeKey :=  uuid.New().String()
	var sessionIDBytes []byte
	sessionIDBytes, err = hub.option.Security.Encrypt([]byte(storeKey), hub.option.SecurityKey) ; if err != nil {
		return
	}
	sessionID =  string(sessionIDBytes)
	err = hub.store.InitSession(ctx, storeKey, hub.option.SessionTTL) ; if err != nil {
		return
	}
	return sessionID, nil
}

func (hub Hub) GetSessionBySessionID(ctx context.Context, sessionID string) (session Session, expired bool, err error) {
	session, has, err := hub.getSession(ctx, sessionID, nil) ; if err != nil {
	    return
	}
	// GetSessionBySessionID 的场景一般是微信小程序或 app
	// 这种场景当 key 错误应当返回key已过期 ，便于调用方告知客户端需要重新登录
	// 过期 = key 不存在
	expired = (has == false)
	return
}
func (hub Hub) getSession(ctx context.Context, sessionID string, writer http.ResponseWriter) (session Session, has bool, err error) {
	var storeKey string
	if sessionID == "" {
		return Session{}, false,&ErrEmptySessionID{}
	}
	storeKeyBytes, err := hub.option.Security.Decrypt([]byte(sessionID), hub.option.SecurityKey) ; if err != nil {
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
	return
}

func (hub Hub) setSessionIDInCookie(writer http.ResponseWriter, sessionID string) {
	http.SetCookie(writer, &http.Cookie{
		Name:       hub.option.Cookie.Name,
		Value:      sessionID,
		Path:       hub.option.Cookie.Path,
		Domain:     hub.option.Cookie.Domain,
		MaxAge:     hub.option.Cookie.MaxAge,
		Secure:     hub.option.Cookie.Secure,
		HttpOnly:   true,
	})
}
func (hub Hub) GetSessionByCookie(ctx context.Context, writer http.ResponseWriter, request *http.Request) (Session, error) {
	var noCookie bool
	cookie, err := request.Cookie(hub.option.Cookie.Name) ; if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			noCookie = true
		} else {
			return Session{}, err
		}
	}
	var sessionID string
	if noCookie {
		sessionID, err = hub.NewSessionID(ctx) ; if err != nil {
			return Session{}, err
		}
		hub.setSessionIDInCookie(writer, sessionID)
	} else {
		sessionID = cookie.Value
	}
	session, has, err := hub.getSession(ctx, sessionID, writer)
	// session 如果过期和恶意攻击的情况 会 has == false (可以运行中清楚 store 的数据以测试这种情况 redis flushdb)
	if has == false {
		// 这种两种情况都生成新的 session
		sessionID, err :=  hub.NewSessionID(ctx) ; if err != nil {
		    return Session{},err
		}
		// 生成新的 sessionID 后再返回 Session{}
		session, _, err = hub.getSession(ctx, sessionID, writer) ; if err != nil {
		    return Session{},err
		}
		// 同时更新客户端cookie
		hub.setSessionIDInCookie(writer, sessionID)
	}
	return session, nil
}