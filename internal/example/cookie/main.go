package main

import (
	"fmt"
	"github.com/go-redis/redis/v8"
	sess "github.com/goclub/session"
	"log"
	"net/http"
	"strconv"
	"time"
)



// 为了便于理解，演示代码中使用 panic 粗糙的处理错误
// 更好的方法：https://github.com/goclub/error
func HandleError(err error) {
	if err != nil {
		panic(err)
	}
}
// 为了便于理解，简化实现
// 更好的方法：https://github.com/goclub/http
func WriteString(w http.ResponseWriter, s string) {
	_, err := w.Write([]byte(s)) ; if err != nil {
		w.WriteHeader(500)
		log.Print(err)
	}
}
func main() {
	redisStore := sess.NewRedisStore(sess.RedisStoreOption{
		Client: redis.NewClient(&redis.Options{
			Network: "tcp",
			Addr: "127.0.0.1:6379",
		}),
		StoreKeyPrefix: "project_name",
	})
	// 线上环境不要使用这里的 key, 应当读取配置文件或配置中心的key
	secureKey := []byte("e9a2f9cbfab74abaa472ff7385dd8224")
	if len(secureKey) != 32 {
		panic("secureKey length must be 32")
	}
	sessHub := sess.NewHub(redisStore, sess.HubOption{
		SecureKey: secureKey,
		Cookie:      sess.HubOptionCookie{
			Name: "project_name_session",
		},
		Security:    sess.DefaultSecurity{},
		SessionTTL:  2 * time.Hour,
	})
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		ctx := request.Context()
		session, err := sessHub.GetSessionByCookie(ctx, writer, request) ; HandleError(err)
		query := request.URL.Query()
		switch query.Get("kind") {
		case "id":
			WriteString(writer, session.ID())
			return
		case "get":
			value, hasValue, err := session.Get(ctx, "name") ; HandleError(err)
			WriteString(writer, fmt.Sprintf("value: %s hasValue: %b",value, hasValue))
			return
		case "set":
			value := "nimo" + strconv.FormatInt(int64(time.Now().Second()), 10)
			err := session.Set(ctx, "name", value) ; HandleError(err)
		case "ttl":
			ttl, err := session.SessionRemainingTTL(ctx) ; HandleError(err)
			WriteString(writer, ttl.String())
			return
		case "delete":
			err := session.Delete(ctx, "name") ; HandleError(err)
		case "destroy":
			err := session.Destroy(ctx) ; HandleError(err)
		default:
			WriteString(writer, `
				<a href="?kind=id">id</a>
				<a href="?kind=get">get</a>
				<a href="?kind=set">set</a>
				<a href="?kind=ttl">ttl</a>
				<a href="?kind=delete">delete</a>
				<a href="?kind=destroy">destroy</a>
			`)
			return
		}
		WriteString(writer, "ok")
	})
	addr := ":2222"
	log.Print("http://127.0.0.1" + addr)
	log.Print(http.ListenAndServe(addr, nil))
}
