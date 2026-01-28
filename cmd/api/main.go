package main

import (
	"log"
	"os"

	"marketcontrol/internal/routes"
	"marketcontrol/pkg/config"
	// "marketcontrol/internal/services"
)

func main() {
	// Initialize database
	config.InitDB()

	// Initialize RabbitMQ (optional, will log warning if not configured)
	if os.Getenv("RABBITMQ_HOST") != "" {
		config.InitRabbitMQ()
		defer func() {
			if config.RabbitMQ != nil {
				config.RabbitMQ.Close()
			}
		}()
		log.Println("RabbitMQ initialized successfully")
	} else {
		log.Println("RabbitMQ not configured, skipping initialization")
	}

	// // Create a new consumer
	// consumer, err := consumer.NewConsumer("your_queue_name")
	// if err != nil {
	//     log.Fatal("Create consumer failed:", err)
	// }
	// defer consumer.Close()

	// // Start consuming messages
	// go func() {
	//     err := consumer.Consume(func(msg []byte) error {
	//         // handle the message here
	//         log.Printf("recv: %s", string(msg))
	//         return nil
	//     })
	//     if err != nil {
	//         log.Printf("msg error: %v", err)
	//     }
	// }()

	// Set up router
	r := routes.SetupRouter()

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
