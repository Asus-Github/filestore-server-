package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"filestore-server/common"
	"filestore-server/util"
)

// Resp : 输出统一格式的json响应 {code, msg, data}
func Resp(c *gin.Context, httpStatus int, code int32, msg string, data interface{}) {
	c.JSON(httpStatus, util.NewRespMsg(int(code), msg, data))
}

// OK : 成功响应, code=10000
func OK(c *gin.Context, msg string, data interface{}) {
	Resp(c, http.StatusOK, common.StatusOK, msg, data)
}

// Fail : 业务失败响应(http状态码200, 由code区分错误类型)
func Fail(c *gin.Context, code int32, msg string) {
	Resp(c, http.StatusOK, code, msg, nil)
}

// AbortWith : 中断请求并输出错误
func AbortWith(c *gin.Context, httpStatus int, code int32, msg string) {
	c.AbortWithStatusJSON(httpStatus, util.NewRespMsg(int(code), msg, nil))
}
