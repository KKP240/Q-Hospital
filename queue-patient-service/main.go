package main

import (
	"log"
	"os"

	"github.com/KKP240/Q-Hospital/auth"
	"github.com/KKP240/Q-Hospital/auth/middleware"
	"github.com/gin-gonic/gin"
	amqp "github.com/rabbitmq/amqp091-go"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func main() {

	var err error

	// ================= JWT =================

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET not set")
	}

	// ================= DATABASE =================

	db, err = gorm.Open(postgres.Open(os.Getenv("DB_URL")), &gorm.Config{})
	failOnError(err, "Database connection failed")

	err = db.AutoMigrate(&Queue{})
	failOnError(err, "Migration failed")

	// ================= RABBITMQ =================

	conn, err := amqp.Dial(os.Getenv("RABBITMQ_URL"))
	failOnError(err, "RabbitMQ connect failed")

	rabbitCh, err = conn.Channel()
	failOnError(err, "Channel open failed")

	err = rabbitCh.ExchangeDeclare(
		"hospital",
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)

	failOnError(err, "Exchange declare failed")

	// start consumer
	startConsumer()

	// ================= API =================

	r := gin.Default()

	// middleware
	authConfig := auth.NewAuthConfig(jwtSecret)

	authorized := r.Group("/")
	authorized.Use(middleware.GinAuthMiddleware(authConfig))
	{
		authorized.GET("/queues", list)
		authorized.PUT("/queues/:id/call", call)
		authorized.PUT("/queues/:id/done", done)
	}

	log.Println("Queue Patient Service running on :8080")

	r.Run(":8080")
}
