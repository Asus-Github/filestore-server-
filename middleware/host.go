package middleware

import (
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// RequestScheme : 获取客户端访问时使用的协议(兼容反向代理)
func RequestScheme(r *http.Request) string {
	if p := r.Header.Get("X-Forwarded-Proto"); p != "" {
		return strings.ToLower(strings.TrimSpace(strings.Split(p, ",")[0]))
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

// RequestHost : 获取客户端访问时使用的host(兼容反向代理)
func RequestHost(r *http.Request) string {
	if h := r.Header.Get("X-Forwarded-Host"); h != "" {
		return strings.TrimSpace(strings.Split(h, ",")[0])
	}
	return r.Host
}

// RequestBaseURL : scheme://host
func RequestBaseURL(c *gin.Context) string {
	return RequestScheme(c.Request) + "://" + RequestHost(c.Request)
}

// ServiceEntry : 计算返回给前端的服务入口
// 配置了LB地址时直接使用; 否则用"当前访问的主机名 + 服务监听端口"拼接(适用于本地/单机部署)
func ServiceEntry(c *gin.Context, lbHost, listenAddr string) string {
	if lbHost != "" {
		if !strings.Contains(lbHost, "://") {
			lbHost = "http://" + lbHost
		}
		return strings.TrimRight(lbHost, "/")
	}
	host := RequestHost(c.Request)
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	if strings.Contains(host, ":") { // ipv6
		host = "[" + host + "]"
	}
	_, port, err := net.SplitHostPort(listenAddr)
	if err != nil || port == "" {
		port = "80"
	}
	return RequestScheme(c.Request) + "://" + host + ":" + port
}
