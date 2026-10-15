package main

import (
	"log"
	"time"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v4"

	_ "filestore-server/common/plugins"
	"filestore-server/config"
	"filestore-server/mq"
	dbproxy "filestore-server/service/dbproxy/client"
	"filestore-server/service/transfer/process"
)

func startTransferService() {
	if !config.AsyncTransferEnable {
		log.Println("异步转移文件功能目前被禁用，请检查相关配置(FS_ASYNC_TRANSFER)")
		return
	}
	log.Println("文件转移服务启动中，开始监听转移任务队列...")

	// 初始化mq client
	mq.Init()

	mq.StartConsume(
		config.TransOSSQueueName,
		"transfer_oss",
		process.Transfer)
}

func main() {
	service := micro.NewService(
		micro.Name("go.micro.service.transfer"), // 服务名称
		micro.RegisterTTL(time.Second*10),       // TTL指定从上一次心跳间隔起，超过这个时间服务会被服务发现移除
		micro.RegisterInterval(time.Second*5),   // 让服务在指定时间内重新注册，保持TTL获取的注册时间有效
		micro.Flags(config.CustomFlags...),
	)
	service.Init(
		micro.Action(func(c *cli.Context) error {
			// 检查是否指定mqhost
			config.ApplyCustomFlags(c)
			return nil
		}),
	)

	// 初始化dbproxy client
	dbproxy.Init(service)

	// 文件转移服务(须在解析命令行参数之后启动, 保证mqhost生效)
	go startTransferService()

	// rpc 服务
	if err := service.Run(); err != nil {
		log.Println(err)
	}
}
