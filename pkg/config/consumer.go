package config

import (
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Consumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   string
}

func NewConsumer(queueName string) (*Consumer, error) {
	ch, err := RabbitMQ.Channel()
	if err != nil {
		return nil, err
	}

	q, err := ch.QueueDeclare(
		queueName,
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		nil,   // args
	)
	if err != nil {
		return nil, err
	}

	return &Consumer{
		conn:    RabbitMQ,
		channel: ch,
		queue:   q.Name,
	}, nil
}

func (c *Consumer) Consume(handler func([]byte) error) error {
	msgs, err := c.channel.Consume(
		c.queue,
		"",    // consumer
		false, // autoAck
		false, // exclusive
		false, // noLocal
		false, // noWait
		nil,   // args
	)
	if err != nil {
		return err
	}

	forever := make(chan bool)

	go func() {
		for msg := range msgs {
			if err := handler(msg.Body); err != nil {
				log.Printf("Handle msg failed: %v", err)
				msg.Nack(false, true) // requeue the message
			} else {
				msg.Ack(false) // successfully processed the message
			}
		}
	}()

	log.Printf("Consumer is running... the port is: %s", c.queue)
	<-forever

	return nil
}

func (c *Consumer) Close() error {
	if err := c.channel.Close(); err != nil {
		return err
	}
	return nil
}
