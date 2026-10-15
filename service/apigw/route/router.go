package route

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"filestore-server/middleware"
	userProto "filestore-server/service/account/proto"
	"filestore-server/service/apigw/handler"
	"filestore-server/static"
)

// noCache : 页面及脚本更新后避免浏览器使用旧缓存
func noCache() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache")
		c.Next()
	}
}

// Router : 网关api路由
func Router(userCli userProto.UserService) *gin.Engine {
	router := gin.Default()
	h := &handler.Handler{UserCli: userCli}

	// 静态文件已通过 go:embed 打包到可执行文件中
	router.Group("/static", noCache()).StaticFS("/", http.FS(static.FS))

	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/static/view/home.html")
	})
	router.GET("/favicon.ico", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	router.GET("/healthz", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// 注册
	router.GET("/user/signup", handler.SignupHandler)
	router.POST("/user/signup", h.DoSignupHandler)
	// 登录
	router.GET("/user/signin", handler.SigninHandler)
	router.POST("/user/signin", h.DoSigninHandler)

	// 以下接口均需要校验token
	auth := router.Group("/", middleware.Authorize(userCli))
	// 退出登录
	auth.POST("/user/signout", h.SignoutHandler)
	// 用户查询
	auth.POST("/user/info", h.UserInfoHandler)
	// 用户文件查询
	auth.POST("/file/query", h.FileQueryHandler)
	// 用户文件修改(重命名)
	auth.POST("/file/update", h.FileMetaUpdateHandler)
	// 用户文件删除
	auth.POST("/file/delete", h.FileDeleteHandler)

	return router
}
