package config

import (
	"fmt"
	"log"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

var RabbitMQ *amqp.Connection

// InitRabbitMQ RabbitMQ with retry logic
func InitRabbitMQ() {
	url := fmt.Sprintf("amqp://%s:%s@%s:%s/",
		os.Getenv("RABBITMQ_USER"),
		os.Getenv("RABBITMQ_PASSWORD"),
		os.Getenv("RABBITMQ_HOST"),
		os.Getenv("RABBITMQ_PORT"),
	)

	maxRetries := 10
	retryDelay := 3 * time.Second

	var conn *amqp.Connection
	var err error

	for i := 0; i < maxRetries; i++ {
		conn, err = amqp.Dial(url)
		if err == nil {
			RabbitMQ = conn
			log.Printf("Successfully connected to RabbitMQ at %s", os.Getenv("RABBITMQ_HOST"))
			return
		}

		if i < maxRetries-1 {
			log.Printf("Failed to connect to RabbitMQ (attempt %d/%d): %v. Retrying in %v...", i+1, maxRetries, err, retryDelay)
			time.Sleep(retryDelay)
		}
	}

	log.Fatalf("Failed to connect to RabbitMQ after %d attempts: %v", maxRetries, err)
}

// DeleteQueue deletes a RabbitMQ queue by name
// If the queue doesn't exist, it will return an error
func DeleteQueue(queueName string) error {
	if RabbitMQ == nil {
		return fmt.Errorf("RabbitMQ connection not initialized")
	}

	ch, err := RabbitMQ.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}
	defer ch.Close()

	// Delete queue
	// ifUnUsed: only delete if queue has no consumers
	// ifEmpty: only delete if queue is empty
	// noWait: don't wait for server response
	_, err = ch.QueueDelete(
		queueName, // queue name
		false,     // ifUnUsed
		false,     // ifEmpty
		false,     // noWait
	)
	if err != nil {
		return fmt.Errorf("failed to delete queue %s: %w", queueName, err)
	}

	log.Printf("Successfully deleted RabbitMQ queue: %s", queueName)
	return nil
}

// PurgeQueue removes all messages from a queue without deleting the queue itself
func PurgeQueue(queueName string) error {
	if RabbitMQ == nil {
		return fmt.Errorf("RabbitMQ connection not initialized")
	}

	ch, err := RabbitMQ.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}
	defer ch.Close()

	// Purge queue (remove all messages)
	_, err = ch.QueuePurge(
		queueName, // queue name
		false,     // noWait
	)
	if err != nil {
		return fmt.Errorf("failed to purge queue %s: %w", queueName, err)
	}

	log.Printf("Successfully purged RabbitMQ queue: %s", queueName)
	return nil
}
