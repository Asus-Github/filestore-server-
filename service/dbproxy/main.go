package main

import (
	"log"
	"time"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v4"

	_ "filestore-server/common/plugins"
	"filestore-server/config"
	dbConn "filestore-server/service/dbproxy/conn"
	dbProxy "filestore-server/service/dbproxy/proto"
	dbRpc "filestore-server/service/dbproxy/rpc"
)

func startRpcService() {
	service := micro.NewService(
		micro.Name("go.micro.service.dbproxy"), // 在注册中心中的服务名称
		micro.RegisterTTL(time.Second*10),      // 声明超时时间, 避免consul不主动删掉已失去心跳的服务节点
		micro.RegisterInterval(time.Second*5),
		micro.Flags(config.CustomFlags...),
	)

	service.Init(
		micro.Action(func(c *cli.Context) error {
			// 检查是否指定dbhost等参数
			config.ApplyCustomFlags(c)
			return nil
		}),
	)

	// 初始化db connection
	if err := dbConn.InitDBConn(); err != nil {
		log.Fatal(err)
	}

	if err := dbProxy.RegisterDBProxyServiceHandler(service.Server(), new(dbRpc.DBProxy)); err != nil {
		log.Fatal(err)
	}
	if err := service.Run(); err != nil {
		log.Println(err)
	}
}

func main() {
	startRpcService()
}
