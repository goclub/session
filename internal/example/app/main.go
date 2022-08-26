package main

import (
	"fmt"
	"github.com/go-redis/redis/v8"
	red "github.com/goclub/redis"
	sess "github.com/goclub/session"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"
)

// 为了便于理解，演示代码中使用 %+v 粗糙的处理错误,这样可能会暴露一下安全信息
// 更好的方法：https://github.com/goclub/error
func HandleError(w http.ResponseWriter, err error) {
	if w == nil && err != nil {
		panic(err)
	}
	if err != nil {
		_, err := fmt.Fprintf(w, "%+v", err)
		if err != nil {
			panic(err)
		}
	}
}

// 为了便于理解，简化实现
// 更好的方法：https://github.com/goclub/http
func WriteString(w http.ResponseWriter, s string) {
	_, err := w.Write([]byte(s))
	if err != nil {
		w.WriteHeader(500)
		log.Print(err)
	}
}
func main() {
	redisStore := sess.NewRedisStore(sess.RedisStoreOption{
		Client: red.NewGoRedisV8(redis.NewClient(&redis.Options{
			Network: "tcp",
			Addr:    "127.0.0.1:6379",
		})),
		StoreKeyPrefix: "project_session_name",
	})
	// 线上环境不要使用 TemporarySecretKey 应当读取配置文件或配置中心的key
	secureKey := sess.TemporarySecretKey()
	sessHub, err := sess.NewHub(redisStore, sess.HubOption{
		SecureKey: secureKey,
		Cookie: sess.HubOptionCookie{
			Name: "project_name_session_cookie",
		},
		Security:   sess.DefaultSecurity{},
		SessionTTL: 2 * time.Hour,
	})
	HandleError(nil, err)
	html, err := ioutil.ReadFile(path.Join(os.Getenv("GOPATH"), "src/github.com/goclub/session/internal/example/app/index.html"))
	HandleError(nil, err)
	http.HandleFunc("/login", func(writer http.ResponseWriter, request *http.Request) {
		ctx := request.Context()
		sessionID, err := sessHub.NewSessionID(ctx)
		HandleError(writer, err)
		WriteString(writer, sessionID)
		return
	})
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		ctx := request.Context()
		// 为了便于演示，通过 query 传递 sessionID ,实际开发中应该在 request.Body(json) 或 header 传递 sessionID
		query := request.URL.Query()
		kind := query.Get("kind")
		// 渲染测试用 html
		if kind == "" {
			WriteString(writer, string(html))
			return
		}
		sessionID := query.Get("sessionID")
		session, sessionExpired, err := sessHub.GetSessionBySessionID(ctx, sessionID)
		HandleError(writer, err)
		if sessionExpired {
			WriteString(writer, "session 已过期，请访问\n/login\n 重新获取")
			return
		}
		switch query.Get("kind") {
		case "id":
			WriteString(writer, session.ID())
			return
		case "get":
			value, hasValue, err := session.Get(ctx, "name")
			HandleError(writer, err)
			WriteString(writer, fmt.Sprintf("value: %s hasValue: %v", value, hasValue))
			return
		case "set":
			value := "nimo" + strconv.FormatInt(int64(time.Now().Second()), 10)
			err := session.Set(ctx, "name", value)
			HandleError(writer, err)
		case "ttl":
			ttl, err := session.SessionRemainingTTL(ctx)
			HandleError(writer, err)
			WriteString(writer, ttl.String())
			return
		case "delete":
			err := session.Delete(ctx, "name")
			HandleError(writer, err)
		case "destroy":
			err := session.Destroy(ctx)
			HandleError(writer, err)
		}
		WriteString(writer, "ok")
	})
	addr := ":3333"
	log.Print("http://127.0.0.1" + addr)
	log.Print(http.ListenAndServe(addr, nil))
}
