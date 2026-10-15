package main

import (
	"log"
	"time"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v4"

	_ "filestore-server/common/plugins"
	"filestore-server/config"
	"filestore-server/service/account/handler"
	proto "filestore-server/service/account/proto"
	dbproxy "filestore-server/service/dbproxy/client"
)

func main() {
	service := micro.NewService(
		micro.Name("go.micro.service.user"),
		micro.RegisterTTL(time.Second*10),
		micro.RegisterInterval(time.Second*5),
		micro.Flags(config.CustomFlags...),
	)

	// 初始化service, 解析命令行参数等
	service.Init(
		micro.Action(func(c *cli.Context) error {
			config.ApplyCustomFlags(c)
			return nil
		}),
	)

	// 初始化dbproxy client
	dbproxy.Init(service)

	if err := proto.RegisterUserServiceHandler(service.Server(), new(handler.User)); err != nil {
		log.Fatal(err)
	}
	if err := service.Run(); err != nil {
		log.Println(err)
	}
}
