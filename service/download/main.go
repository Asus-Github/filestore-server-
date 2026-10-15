package main

import (
	"log"
	"time"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v4"

	_ "filestore-server/common/plugins"
	"filestore-server/config"
	userProto "filestore-server/service/account/proto"
	dbproxy "filestore-server/service/dbproxy/client"
	dlProto "filestore-server/service/download/proto"
	"filestore-server/service/download/route"
	dlRpc "filestore-server/service/download/rpc"
)

func startAPIService(service micro.Service) {
	userCli := userProto.NewUserService("go.micro.service.user", service.Client())
	router := route.Router(userCli)
	log.Println("download api listening on " + config.DownloadServiceHost)
	if err := router.Run(config.DownloadServiceHost); err != nil {
		log.Fatal(err)
	}
}

func main() {
	service := micro.NewService(
		micro.Name("go.micro.service.download"), // 在注册中心中的服务名称
		micro.RegisterTTL(time.Second*10),
		micro.RegisterInterval(time.Second*5),
		micro.Flags(config.CustomFlags...),
	)
	service.Init(
		micro.Action(func(c *cli.Context) error {
			config.ApplyCustomFlags(c)
			return nil
		}),
	)

	// 初始化dbproxy client
	dbproxy.Init(service)

	if config.TokenSecret == config.DefaultTokenSecret {
		log.Println("WARNING: FS_TOKEN_SECRET 未设置, 正在使用默认密钥签名下载链接, 生产环境请务必修改")
	}

	// api 服务
	go startAPIService(service)

	// rpc 服务
	if err := dlProto.RegisterDownloadServiceHandler(service.Server(), new(dlRpc.Download)); err != nil {
		log.Fatal(err)
	}
	if err := service.Run(); err != nil {
		log.Println(err)
	}
}
