package route

import (
	"github.com/gin-gonic/gin"

	"filestore-server/middleware"
	userProto "filestore-server/service/account/proto"
	"filestore-server/service/download/api"
)

// Router : 路由表配置
func Router(userCli userProto.UserService) *gin.Engine {
	// gin framework, 包括Logger, Recovery
	router := gin.Default()

	// 使用gin插件支持跨域请求(须在鉴权之前, 保证预检请求OPTIONS能通过)
	router.Use(middleware.CORS())

	router.GET("/healthz", func(c *gin.Context) {
		c.String(200, "ok")
	})

	// 文件下载: 支持签名链接或登录token两种鉴权方式, 由handler内部校验
	download := func(c *gin.Context) {
		api.DownloadHandler(c, func(c *gin.Context) (string, bool) {
			return middleware.VerifyRequest(c, userCli)
		})
	}
	router.GET("/file/download", download)
	router.HEAD("/file/download", download)

	// 获取下载链接需要登录
	router.POST("/file/downloadurl", middleware.Authorize(userCli), api.DownloadURLHandler)

	return router
}
