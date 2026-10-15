package mq

import (
	"log"
	"time"

	"filestore-server/config"
)

var done = make(chan struct{})

// StartConsume : 接收消息, 处理成功才ack; 处理失败的消息转入错误队列待后续处理
// 连接断开时自动重连并继续消费, 直到调用StopConsume
func StartConsume(qName, cName string, callback func(msg []byte) bool) {
	for {
		select {
		case <-done:
			return
		default:
		}

		ch, err := getChannel()
		if err != nil {
			log.Println("rabbitmq connect failed, retry in 3s:", err)
			time.Sleep(3 * time.Second)
			continue
		}
		// 每次只预取一条, 避免单个消费者积压大量未确认消息
		if err := ch.Qos(1, 0, false); err != nil {
			log.Println(err)
		}
		msgs, err := ch.Consume(
			qName,
			cName,
			false, // 手动应答
			false, // 非唯一的消费者
			false, // rabbitMQ只能设置为false
			false, // noWait
			nil)
		if err != nil {
			log.Println("rabbitmq consume failed, retry in 3s:", err)
			time.Sleep(3 * time.Second)
			continue
		}

	loop:
		for {
			select {
			case <-done:
				return
			case d, ok := <-msgs:
				if !ok {
					// channel被关闭, 重新建立连接
					break loop
				}
				if callback(d.Body) {
					d.Ack(false)
					continue
				}
				// 处理失败: 写入错误队列后确认原消息; 写入失败则重新入队
				if Publish(config.TransExchangeName, config.TransOSSErrRoutingKey, d.Body) {
					d.Ack(false)
				} else {
					d.Nack(false, true)
					time.Sleep(time.Second)
				}
			}
		}
	}
}

// StopConsume : 停止监听队列
func StopConsume() {
	select {
	case <-done:
	default:
		close(done)
	}
}
