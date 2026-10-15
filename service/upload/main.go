package main

import (
	"log"
	"os"
	"time"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v4"

	rPool "filestore-server/cache/redis"
	_ "filestore-server/common/plugins"
	"filestore-server/config"
	"filestore-server/mq"
	userProto "filestore-server/service/account/proto"
	dbproxy "filestore-server/service/dbproxy/client"
	"filestore-server/service/upload/api"
	upProto "filestore-server/service/upload/proto"
	"filestore-server/service/upload/route"
	upRpc "filestore-server/service/upload/rpc"
)

func startAPIService(service micro.Service) {
	userCli := userProto.NewUserService("go.micro.service.user", service.Client())
	router := route.Router(userCli)
	log.Println("upload api listening on " + config.UploadServiceHost)
	if err := router.Run(config.UploadServiceHost); err != nil {
		log.Fatal(err)
	}
}

func main() {
	service := micro.NewService(
		micro.Name("go.micro.service.upload"), // 服务名称
		micro.RegisterTTL(time.Second*10),     // TTL指定从上一次心跳间隔起，超过这个时间服务会被服务发现移除
		micro.RegisterInterval(time.Second*5), // 让服务在指定时间内重新注册，保持TTL获取的注册时间有效
		micro.Flags(config.CustomFlags...),
	)
	service.Init(
		micro.Action(func(c *cli.Context) error {
			// 检查是否指定mqhost/cachehost等参数
			config.ApplyCustomFlags(c)
			return nil
		}),
	)

	for _, dir := range []string{config.TempLocalRootDir, config.TempPartRootDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("create dir %s failed: %v", dir, err)
		}
	}
	if err := rPool.Ping(); err != nil {
		log.Println("WARNING: redis is unavailable, multipart upload will not work:", err)
	}

	// 初始化dbproxy client
	dbproxy.Init(service)
	// 初始化mq client
	mq.Init()

	// api 服务
	go startAPIService(service)
	// 清理过期的分块
	go api.CleanExpiredParts()

	// rpc 服务
	if err := upProto.RegisterUploadServiceHandler(service.Server(), new(upRpc.Upload)); err != nil {
		log.Fatal(err)
	}
	if err := service.Run(); err != nil {
		log.Println(err)
	}
}
