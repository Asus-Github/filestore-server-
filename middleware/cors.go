package middleware

import (
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	cfg "filestore-server/config"
)

// CORS : 上传/下载服务与网关页面不同源, 需要允许跨域请求
// 鉴权使用 Authorization 头而非cookie, 因此不需要 AllowCredentials
func CORS() gin.HandlerFunc {
	conf := cors.Config{
		AllowMethods:  []string{"GET", "HEAD", "POST", "OPTIONS"},
		AllowHeaders:  []string{"Origin", "Range", "If-Range", "Authorization", "X-Requested-With", "Content-Type"},
		ExposeHeaders: []string{"Content-Length", "Accept-Ranges", "Content-Range", "Content-Disposition"},
		MaxAge:        12 * time.Hour,
	}
	origins := []string{}
	for _, o := range strings.Split(cfg.CORSAllowOrigins, ",") {
		if o = strings.TrimSpace(o); o != "" {
			origins = append(origins, o)
		}
	}
	if len(origins) == 0 || (len(origins) == 1 && origins[0] == "*") {
		conf.AllowAllOrigins = true
	} else {
		conf.AllowOrigins = origins
	}
	return cors.New(conf)
}
