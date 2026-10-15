package mq

import (
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"filestore-server/config"
)

var (
	mu      sync.Mutex
	conn    *amqp.Connection
	channel *amqp.Channel
)

// Init : 初始化MQ连接信息, 并声明转移所需的exchange/queue
func Init() {
	// 是否开启异步转移功能，开启时才初始化rabbitMQ连接
	if !config.AsyncTransferEnable {
		return
	}
	if _, err := getChannel(); err != nil {
		log.Println("rabbitmq init failed:", err)
	}
}

// getChannel : 获取可用的channel, 连接断开时自动重连
func getChannel() (*amqp.Channel, error) {
	mu.Lock()
	defer mu.Unlock()

	if channel != nil && !channel.IsClosed() && conn != nil && !conn.IsClosed() {
		return channel, nil
	}
	if conn != nil && !conn.IsClosed() {
		conn.Close()
	}
	conn, channel = nil, nil

	c, err := amqp.DialConfig(config.RabbitURL, amqp.Config{Heartbeat: 10 * time.Second})
	if err != nil {
		return nil, err
	}
	ch, err := c.Channel()
	if err != nil {
		c.Close()
		return nil, err
	}
	if err := declareTopology(ch); err != nil {
		c.Close()
		return nil, err
	}
	conn, channel = c, ch
	return ch, nil
}

// declareTopology : 声明exchange及队列(幂等), 避免消息因队列不存在被丢弃
func declareTopology(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(config.TransExchangeName, "direct", true, false, false, false, nil); err != nil {
		return err
	}
	queues := map[string]string{
		config.TransOSSQueueName:    config.TransOSSRoutingKey,
		config.TransOSSErrQueueName: config.TransOSSErrRoutingKey,
	}
	for q, key := range queues {
		if _, err := ch.QueueDeclare(q, true, false, false, false, nil); err != nil {
			return err
		}
		if err := ch.QueueBind(q, key, config.TransExchangeName, false, nil); err != nil {
			return err
		}
	}
	return nil
}
