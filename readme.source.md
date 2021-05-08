# session

> 安全易用的 session golang 库

## 特色

1. 支持多种 session 传输载体（ccokie, header,request body）
2. 支持多种存储服务（redis  mysql ...）
3. 包含从 0 开始设计并实现 session 的教程

## 示例

**使用 cookie 自动传递 session **

[cookie|embed](./internal/example/cookie/main.go)

**前端手动传递 session**

[app|embed](./internal/example/app/main.go)


**前端使用 header 半自动传递 session**

[app_header|embed](./internal/example/app_header/main.go)


## 理论与实现

### 用户凭证

> HTTP 请求都是无状态的，但是我们的 Web 应用通常都需要知道发起请求的人是谁。为了解决这个问题，HTTP 协议设计了一个特殊的请求头: [Cookie](https://zh.wikipedia.org/wiki/Cookie)。服务端可以通过响应头（set-cookie）将少量数据响应给客户端，浏览器会遵循协议将数据保存，并在下次请求同一个服务的时候带上（浏览器也会遵循协议，只在访问符合 Cookie 指定规则的网站时带上对应的 Cookie 来保证安全性）。

如果直接将 userid 保存在 cookie 中，虽然能实现用户标识，但是是不安全的。因为恶意攻击者可以在 cookie 中设置其他用户的的 userid，以达到登录别人账户的目的。

为了提高安全性使用 session 机制实现用户标识：

1. 服务器端生成一个不可猜测的字符串作为 `SessionID`
2. 根据 `SessionID`在 redis 中创建一个 [hashes](http://www.redis.cn/topics/data-types-intro.html#hashes)
3. 将 `SessionID` 通过 http header [set-cookie](https://developer.mozilla.org/zh-CN/docs/Web/HTTP/Cookies#%E5%88%9B%E5%BB%BAcookie) 传递给客户端
4. 浏览器接收到 set-cookie  后将 cookie 保存在浏览器
5. 接下来浏览器向服务器发起请求时会附带 cookie
6. 对于查询请求服务器读取 cookie 中的 SessionID 并查找到 redis 中的 hashes
7. 对于登录请求服务器设置执行 redis: hset key feild value

> 读取 cookie 的方法是 request.Cookie()
> 设置 cookie 的方法是 http.SetCookie()
> 可参考 sess.CookieReadWriter{}

> 感兴趣的可以自己根据上述流程自己实现一个 session，再回来继续看。

> 当实现了基于 cookie 的 session 之后需要考虑在 微信小程序和app 的场景下不能像 web 一样便捷的使用 cookie 传递 SessionID。
> 可先自行考虑如何解决这个问题，随后查看 [app](./internal/example/app/main.go)

### 安全性

如果将生成的 uuid 直接作为 redis hashes 的 key 和 SessionID 是不安全的。虽然理论上 uuid 不可猜测，但还是应该加一层。[AES](https://cn.bing.com/search?q=aes)双向加密和[BASE64](https://cn.bing.com/search?q=BASE64)编码

这样客户端拿到的 sessionID 大概是这样的：

```
// session id
aHhndHZjZHpxaXZ4enllemTygd0GQUyhFmJEzJZQhkvqenxZ655iNyOp5o380VAIBUDP5X5NLCbXOfixdEx8SA==
```

而解码后大概是这样的

```
// store key
ab883938-f878-4d25-a528-b72a09b7de3f
```

使用 uuid 作为redis hashes 的 key，aes+base64 加密后的字符串作为 sessionID。这样就增加了安全性，恶意攻击者在没有加密秘钥的情况下无法轻易猜测 redis 中的 key。

> set cookie 时一定要打开 [HttpOnly](https://cn.bing.com/search?q=httponly)

### 有效期

为了进一步提高安全性，避免用户 sessionID 被攻击者获取导致的安全问题。需要给每个 session 设置一个有效期。例如一小时。

这个实现比较简单，在 `NewSessionID()` 时使用 redis 的 ttl 机制实现即可。

但为了进一步提高用户体验，在用户短时间内一直在与服务器进行交互时候需实现自动续期功能。

在每次接收到用户请求的 SessionID 并转换成 StoreKey 之后，检查 redis key 剩余的有效期，如果有效期超过30分钟（1h/2）则再次设置 ttl 一小时.


### 有效性

如果恶意攻击者先登录系统，拿到一个 SessionID ，然后在 session 已经过期后再次使用此 SessionID 进行访问。一般情况下这种恶意攻击不会产生什么问题。

但是为了安全性考虑应当在每次 sessionID 解密为 storeKey 后redis exists storeKey 验证数据是否存在。

如果数据不存在则视为数据可能是过期和恶意攻击。这种情况下如果直接服务器返回错误，会误伤一些session过期的用户。**可以在 store key** 不存在时生成新的 SessionID 并 set-cookie 设置到客户端的 cookie 中.


### 接口设计

当实现了上述功能后，需要封装代码。将代码分为三层

1. `http` API协议层
2. `session` 逻辑层
3. `store` 数据存储层

文字难以表达，建议使用一段时间 goclub/session 。然后阅读 goclub/session 的源码帮助理解。
