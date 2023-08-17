package testSess

import (
	"context"
	sess "github.com/goclub/session"
	"github.com/stretchr/testify/assert"
	"net/http/httptest"
	"testing"
)

func TestCookie(t *testing.T, store sess.Store, option sess.HubOption) {
	ctx := context.Background()
	hub, err := sess.NewHub(store, option)
	assert.NoError(t, err)
	request := httptest.NewRequest("GET", "/", nil)
	writer := httptest.NewRecorder()
	session, err := hub.GetSessionByCookie(ctx, writer, request)
	assert.NoError(t, err)
	sessionID := session.ID()
	{
		value, has, err := session.Get(ctx, "name")
		assert.NoError(t, err)
		assert.Equal(t, value, "")
		assert.Equal(t, has, false)
	}
	{
		err := session.Set(ctx, "name", "nimo")
		assert.NoError(t, err)
	}
	{
		value, has, err := session.Get(ctx, "name")
		assert.NoError(t, err)
		assert.Equal(t, value, "nimo")
		assert.Equal(t, has, true)
	}
	setCookie := writer.Header().Get("set-cookie")
	assert.Contains(t, setCookie, option.Cookie.Name+"="+sessionID)
	newRequest := httptest.NewRequest("get", "/set", nil)
	newRequest.Header.Set("cookie", setCookie)
	newWriter := httptest.NewRecorder()
	{
		session, err := hub.GetSessionByCookie(ctx, newWriter, newRequest)
		value, has, err := session.Get(ctx, "name")
		assert.NoError(t, err)
		assert.Equal(t, value, "nimo")
		assert.Equal(t, has, true)
	}
}
