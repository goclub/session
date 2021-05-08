package main

import (
	"fmt"
	"github.com/go-redis/redis/v8"
	sess "github.com/goclub/session"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
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
		Header: sess.HubOptionHeader{
			Key: "token",
		},
		Security:    sess.DefaultSecurity{},
		SessionTTL:  2 * time.Hour,
	})
	html, err := ioutil.ReadFile(path.Join(os.Getenv("GOPATH"), "src/github.com/goclub/session/internal/example/app_header/index.html")) ; HandleError(err)
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
		// 使用 GetSessionByHeader 时，前端需要对 header 做同步处理
		// 前端请参考 $GOPATH/src/github.com/goclub/session/internal/example/app_header/index.html
		session, err := sessHub.GetSessionByHeader(ctx, writer, request.Header) ; HandleError(err)
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
		}
		WriteString(writer, "ok")
	})
	addr := ":4444"
	log.Print("http://127.0.0.1" + addr)
	log.Print(http.ListenAndServe(addr, nil))
}
