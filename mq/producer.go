package mq

import (
	"context"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Publish : 发布消息(持久化), 失败时返回false
func Publish(exchange, routingKey string, msg []byte) bool {
	ch, err := getChannel()
	if err != nil {
		log.Println("rabbitmq publish failed:", err)
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = ch.PublishWithContext(ctx,
		exchange,
		routingKey,
		true,  // mandatory: 没有对应的queue时消息退回而不是静默丢弃
		false, // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         msg,
		})
	if err != nil {
		log.Println("rabbitmq publish failed:", err)
		return false
	}
	return true
}
