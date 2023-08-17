package testSess

import (
	"context"
	"github.com/go-redis/redis/v8"
	red "github.com/goclub/redis"
	sess "github.com/goclub/session"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestTestCookie(t *testing.T) {
	ctx := context.Background()
	client := redis.NewClient(&redis.Options{
		Network: "tcp",
		Addr:    "127.0.0.1:6379",
	})
	assert.NoError(t, client.Ping(ctx).Err())
	redisStore := sess.NewRedisStore(sess.RedisStoreOption{
		Client:         red.NewGoRedisV8(client),
		StoreKeyPrefix: "project_session_name",
	})
	TestCookie(t, redisStore, sess.HubOption{
		// SecurityKey len must be 32
		SecureKey: []byte("e9a2f9cbfab74abaa472ff7385dd8224"),
		Cookie: sess.HubOptionCookie{
			Name: "project_name_session_cookie",
		},
		Security:   sess.DefaultSecurity{},
		SessionTTL: time.Hour * 1,
	})
}
