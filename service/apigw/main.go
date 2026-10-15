package main

import (
	"log"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v4"

	"github.com/go-micro/plugins/v4/wrapper/breaker/hystrix"
	"github.com/go-micro/plugins/v4/wrapper/ratelimiter/ratelimit"
	ratelimit2 "github.com/juju/ratelimit"

	_ "filestore-server/common/plugins"
	"filestore-server/config"
	userProto "filestore-server/service/account/proto"
	"filestore-server/service/apigw/route"
)

func init() {
	// hystrix默认超时为1秒、并发上限10, 登录时的bcrypt计算及并发请求容易误触发熔断
	hystrix.ConfigureDefault(hystrix.CommandConfig{
		Timeout:               5000,
		MaxConcurrentRequests: 200,
		ErrorPercentThreshold: 50,
	})
}

func main() {
	// 配置请求容量及qps
	bRate := ratelimit2.NewBucketWithRate(1000, 2000)
	service := micro.NewService(
		micro.Name("go.micro.api.apigw"),
		micro.Flags(config.CustomFlags...),
		// 加入限流功能, false为不等待(超限即返回请求失败)
		micro.WrapClient(ratelimit.NewClientWrapper(bRate, false)),
		// 加入熔断功能, 处理rpc调用失败的情况(circuit breaker)
		micro.WrapClient(hystrix.NewClientWrapper()),
	)
	// 初始化, 解析命令行参数等
	service.Init(
		micro.Action(func(c *cli.Context) error {
			config.ApplyCustomFlags(c)
			return nil
		}),
	)

	// 初始化一个account服务的客户端
	userCli := userProto.NewUserService("go.micro.service.user", service.Client())

	r := route.Router(userCli)
	log.Println("apigw listening on " + config.APIGWServiceHost)
	if err := r.Run(config.APIGWServiceHost); err != nil {
		log.Fatal(err)
	}
}
