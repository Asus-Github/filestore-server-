package route

import (
	"github.com/gin-gonic/gin"

	"filestore-server/middleware"
	userProto "filestore-server/service/account/proto"
	"filestore-server/service/upload/api"
)

// Router : 路由表配置
func Router(userCli userProto.UserService) *gin.Engine {
	// gin framework, 包括Logger, Recovery
	router := gin.Default()
	// 分块上传的分块大小为MB级, 无需为multipart表单在内存中缓存过多数据
	router.MaxMultipartMemory = 8 << 20

	// 使用gin插件支持跨域请求(须在鉴权之前, 保证预检请求OPTIONS能通过)
	router.Use(middleware.CORS())

	router.GET("/healthz", func(c *gin.Context) {
		c.String(200, "ok")
	})

	// 之后的所有handler都会经过拦截器进行token校验(通过account微服务验证)
	auth := router.Group("/", middleware.Authorize(userCli))

	// 文件上传相关接口
	auth.POST("/file/upload", api.DoUploadHandler)
	// 秒传接口
	auth.POST("/file/fastupload", api.TryFastUploadHandler)

	// 分块上传接口
	auth.POST("/file/mpupload/init", api.InitialMultipartUploadHandler)
	auth.POST("/file/mpupload/uppart", api.UploadPartHandler)
	auth.POST("/file/mpupload/complete", api.CompleteUploadHandler)
	auth.POST("/file/mpupload/cancel", api.CancelUploadHandler)

	return router
}
