package config

import (
	"log"
	"strings"

	"github.com/urfave/cli/v2"
)

// CustomFlags : 自定义命令行参数(为空时使用config中的默认值/环境变量)
var CustomFlags = []cli.Flag{
	&cli.StringFlag{
		Name:  "dbhost",
		Usage: "database address, e.g. 127.0.0.1:3306",
	},
	&cli.StringFlag{
		Name:  "mqhost",
		Usage: "mq(rabbitmq) address, host:port or full amqp:// url",
	},
	&cli.StringFlag{
		Name:  "cachehost",
		Usage: "cache(redis) address, e.g. 127.0.0.1:6379",
	},
	&cli.StringFlag{
		Name:  "cephhost",
		Usage: "ceph rgw endpoint, e.g. http://127.0.0.1:7480",
	},
}

// ApplyCustomFlags : 用命令行参数覆盖默认配置, 需在 service.Init 的 Action 中调用
func ApplyCustomFlags(c *cli.Context) {
	if v := strings.TrimSpace(c.String("dbhost")); v != "" {
		log.Println("custom db address: " + v)
		MySQLHost = withDefaultPort(v, "3306")
	}
	if v := strings.TrimSpace(c.String("cachehost")); v != "" {
		log.Println("custom cache address: " + v)
		RedisHost = withDefaultPort(v, "6379")
	}
	if v := strings.TrimSpace(c.String("mqhost")); v != "" {
		log.Println("custom mq address: " + v)
		UpdateRabbitHost(v)
	}
	if v := strings.TrimSpace(c.String("cephhost")); v != "" {
		log.Println("custom ceph address: " + v)
		if !strings.Contains(v, "://") {
			v = "http://" + v
		}
		CephGWEndpoint = v
	}
}

// withDefaultPort : 地址未指定端口时补上默认端口
func withDefaultPort(addr, port string) string {
	if strings.Contains(addr, ":") {
		return addr
	}
	return addr + ":" + port
}
