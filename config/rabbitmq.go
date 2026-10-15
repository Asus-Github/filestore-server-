package config

import "strings"

const (
	// TransExchangeName : 用于文件transfer的交换机
	TransExchangeName = "uploadserver.trans"
	// TransOSSQueueName : oss转移队列名
	TransOSSQueueName = "uploadserver.trans.oss"
	// TransOSSErrQueueName : oss转移失败后写入另一个队列的队列名
	TransOSSErrQueueName = "uploadserver.trans.oss.err"
	// TransOSSRoutingKey : routingkey
	TransOSSRoutingKey = "oss"
	// TransOSSErrRoutingKey : 转移失败消息的routingkey
	TransOSSErrRoutingKey = "oss.err"
)

var (
	// AsyncTransferEnable : 是否开启文件异步转移(默认同步, FS_ASYNC_TRANSFER=true开启)
	AsyncTransferEnable = envBool("FS_ASYNC_TRANSFER", false)
	// RabbitURL : rabbitmq服务的入口url
	RabbitURL = envString("FS_RABBIT_URL", "amqp://guest:guest@127.0.0.1:5672/")
)

// UpdateRabbitHost : 更新mq地址, 支持完整的 amqp:// url 或 host[:port]
func UpdateRabbitHost(host string) {
	if strings.HasPrefix(host, "amqp://") || strings.HasPrefix(host, "amqps://") {
		RabbitURL = host
		return
	}
	if !strings.Contains(host, ":") {
		host += ":5672"
	}
	RabbitURL = "amqp://guest:guest@" + host + "/"
}
