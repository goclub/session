package sess

import (
	"context"
	"errors"
	xerr "github.com/goclub/error"
	"net/http"
)

type SessionHttpReadWriter interface {
	Read(ctx context.Context, hubOption HubOption) (sessionID string, has bool, err error)
	Write(ctx context.Context, hubOption HubOption, sessionID string) (err error)
	Destroy(ctx context.Context, option HubOption) (err error)
}

func (hub Hub) GetSessionByReadWriter(ctx context.Context, rw SessionHttpReadWriter) (session Session, err error) {
	sessionID, has, err := rw.Read(ctx, hub.option)
	if err != nil {
		return
	}
	// 如果客户端没有session 则生成新的 session
	if has == false {
		sessionID, err = hub.NewSessionID(ctx)
		if err != nil {
			return
		}
		err = rw.Write(ctx, hub.option, sessionID)
		if err != nil {
			return
		}
	}
	var hasSession bool
	session, hasSession, err = hub.getSessionByReadWriter(ctx, sessionID, rw)
	if err != nil {
		return
	}
	// session 如果过期和恶意攻击的情况 会 hasSession == false
	// (可以在已经 NewSessionID 之后清除 store 的数据以测试这种情况,例如 redis flushdb)
	if hasSession == false {
		// 过期和恶意攻击的两种情况都生成新的 session
		sessionID, err := hub.NewSessionID(ctx)
		if err != nil {
			return Session{}, err
		}
		// 生成新的 sessionID 后再返回 Session{}
		session, _, err = hub.getSessionByReadWriter(ctx, sessionID, rw)
		if err != nil {
			return Session{}, err
		}
		// 更新 http sessionID
		err = rw.Write(ctx, hub.option, sessionID)
		if err != nil {
			return Session{}, err
		}
	}
	return session, nil
}

type EmptyHttpReadWirter struct{}

func (EmptyHttpReadWirter) Read(ctx context.Context, hubOption HubOption) (sessionID string, has bool, err error) {
	return
}
func (EmptyHttpReadWirter) Write(ctx context.Context, hubOption HubOption, sessionID string) (err error) {
	return
}
func (EmptyHttpReadWirter) Destroy(ctx context.Context, option HubOption) (err error) { return }

type CookieReadWriter struct {
	Writer  http.ResponseWriter
	Request *http.Request
}

func (rw CookieReadWriter) Read(ctx context.Context, hubOption HubOption) (sessionID string, has bool, err error) {
	if len(hubOption.Cookie.Name) == 0 {
		return "", false, xerr.New("goclub/session: you forget set HubOption{}.Cookie.Name")
	}
	var noCookie bool
	cookie, err := rw.Request.Cookie(hubOption.Cookie.Name)
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			noCookie = true
		} else {
			return "", false, err
		}
	}
	if noCookie {
		return "", false, nil
	}
	return cookie.Value, true, nil
}
func (rw CookieReadWriter) Write(ctx context.Context, hubOption HubOption, sessionID string) (err error) {
	if len(hubOption.Cookie.Name) == 0 {
		return xerr.New("goclub/session: you forget set HubOption{}.Cookie.Name")
	}
	http.SetCookie(rw.Writer, newCookieFromOption(sessionID, hubOption.Cookie))
	return
}

func (rw CookieReadWriter) Destroy(ctx context.Context, hubOption HubOption) (err error) {
	opt := hubOption.Cookie
	opt.MaxAge = -1
	newCookie := newCookieFromOption("", opt)
	http.SetCookie(rw.Writer, newCookie)
	return
}

type HeaderReadWriter struct {
	Writer http.ResponseWriter
	// 故意将 HeaderReadWriter 设计成需要 Header 而不是 *http.Request
	// 目的是避免吧 sessHub.GetSessionByHeader() 当做 sessHub.GetSessionByCookie() 使用
	Header http.Header
}

func (rw HeaderReadWriter) Destroy(ctx context.Context, option HubOption) (err error) {
	if len(option.Header.Key) == 0 {
		return xerr.New("goclub/session: you forget set HubOption{}.Header.Key")
	}
	// Header Destroy 什么都不用做
	return
}

func (rw HeaderReadWriter) Read(ctx context.Context, hubOption HubOption) (sessionID string, has bool, err error) {
	if len(hubOption.Header.Key) == 0 {
		return "", false, xerr.New("goclub/session: you forget set HubOption{}.Header.Key")
	}
	has = true
	sessionID = rw.Header.Get(hubOption.Header.Key)
	if len(sessionID) == 0 {
		return "", false, nil
	}
	return
}
func (rw HeaderReadWriter) Write(ctx context.Context, hubOption HubOption, sessionID string) (err error) {
	if len(hubOption.Header.Key) == 0 {
		return xerr.New("goclub/session: you forget set HubOption{}.Header.Key")
	}
	rw.Writer.Header().Set(hubOption.Header.Key, sessionID)
	return
}
