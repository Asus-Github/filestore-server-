package middleware

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"filestore-server/common"
	userProto "filestore-server/service/account/proto"
)

// CtxUsername : 鉴权通过后, 当前登录用户名在gin.Context中的key
const CtxUsername = "auth.username"

// ExtractToken : 从请求中获取token
// 优先使用 Authorization: Bearer <token>, 兼容旧接口的 query/form 参数 token
func ExtractToken(c *gin.Context) string {
	if h := c.GetHeader("Authorization"); len(h) > 7 && strings.EqualFold(h[:7], "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	if t := c.Query("token"); t != "" {
		return t
	}
	if isURLEncodedForm(c) {
		return c.PostForm("token")
	}
	return ""
}

// claimedUsername : 客户端声明的用户名(可选), 不为空时必须与token归属用户一致
func claimedUsername(c *gin.Context) string {
	if u := c.Query("username"); u != "" {
		return u
	}
	if isURLEncodedForm(c) {
		return c.PostForm("username")
	}
	return ""
}

// isURLEncodedForm : 只解析urlencoded表单, 避免在鉴权阶段把multipart/二进制请求体整个读入内存
func isURLEncodedForm(c *gin.Context) bool {
	return c.Request.Method != http.MethodGet &&
		strings.HasPrefix(c.ContentType(), "application/x-www-form-urlencoded")
}

// VerifyRequest : 校验请求携带的token, 成功时返回token归属的用户名
// 失败时已输出错误响应并中断请求
func VerifyRequest(c *gin.Context, userCli userProto.UserService) (string, bool) {
	token := ExtractToken(c)
	if token == "" {
		AbortWith(c, http.StatusUnauthorized, common.StatusTokenInvalid, "请先登录")
		return "", false
	}
	resp, err := userCli.VerifyToken(c.Request.Context(), &userProto.ReqVerifyToken{
		Username: claimedUsername(c),
		Token:    token,
	})
	if err != nil {
		log.Println("verify token failed:", err)
		AbortWith(c, http.StatusServiceUnavailable, common.StatusServerError, "鉴权服务暂不可用, 请稍后重试")
		return "", false
	}
	if resp.Code != common.StatusOK {
		AbortWith(c, http.StatusUnauthorized, common.StatusTokenInvalid, resp.Message)
		return "", false
	}
	return resp.Username, true
}

// Authorize : http请求拦截器, token校验失败时返回401及 code=10005
func Authorize(userCli userProto.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		username, ok := VerifyRequest(c, userCli)
		if !ok {
			return
		}
		c.Set(CtxUsername, username)
		c.Next()
	}
}

// Username : 获取当前登录的用户名(须在Authorize之后使用)
func Username(c *gin.Context) string {
	return c.GetString(CtxUsername)
}
