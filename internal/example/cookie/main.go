package main

import (
	"fmt"
	"github.com/go-redis/redis/v8"
	red "github.com/goclub/redis"
	sess "github.com/goclub/session"
	"log"
	"net/http"
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
		SessionTTL: 2 * time.Hour,
	})
	HandleError(nil, err)
	http.HandleFunc("/test", func(writer http.ResponseWriter, request *http.Request) {
		ctx := request.Context()
		query := request.URL.Query()
		if query.Get("kind") == "" {
			WriteString(writer, `
				<h1>使用 cookie 自动传递 session id</h1>
				<a href="?kind=id">id</a>
				<a href="?kind=get">get</a>
				<a href="?kind=set">set</a>
				<a href="?kind=ttl">ttl</a>
				<a href="?kind=delete">delete</a>
				<a href="?kind=destroy">destroy</a>
				<hr />
				使用浏览器返回按钮返回页面时记得刷新页面获取最新 cookie 值
				<hr />
			`)
			cookie, err := request.Cookie("project_name_session_cookie")
			cookieDump := fmt.Sprintf("%v<br/>%v", cookie, err)
			WriteString(writer, cookieDump)
			return
		}
		session, err := sessHub.GetSessionByCookie(ctx, writer, request)
		HandleError(writer, err)
		switch query.Get("kind") {
		case "id":
			WriteString(writer, session.ID())
		case "get":
			value, hasValue, err := session.Get(ctx, "name")
			HandleError(writer, err)
			WriteString(writer, fmt.Sprintf("value: %s hasValue: %b", value, hasValue))
		case "set":
			value := "nimo" + strconv.FormatInt(int64(time.Now().Second()), 10)
			err := session.Set(ctx, "name", value)
			HandleError(writer, err)
		case "ttl":
			ttl, err := session.SessionRemainingTTL(ctx)
			HandleError(writer, err)
			WriteString(writer, ttl.String())
		case "delete":
			err := session.Delete(ctx, "name")
			HandleError(writer, err)
		case "destroy":
			err := session.Destroy(ctx)
			HandleError(writer, err)
		}
		WriteString(writer, "ok")
	})
	addr := ":2222"
	log.Print("http://127.0.0.1" + addr + "/test")
	log.Print(http.ListenAndServe(addr, nil))
}
