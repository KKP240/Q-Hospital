package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	amqp "github.com/rabbitmq/amqp091-go"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

// Render Error
func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func main() {
	var err error

	userServiceURL = os.Getenv("USER_SERVICE_URL")
	if userServiceURL == "" {
		userServiceURL = "http://localhost:8084"
	}

	// Database
	db, err = gorm.Open(postgres.Open(os.Getenv("DB_URL")), &gorm.Config{})
	failOnError(err, "Failed to connect to Database")

	err = db.AutoMigrate(&Appointment{})
	failOnError(err, "Migration failed")

	// RabbitMQ
	conn, err := amqp.Dial(os.Getenv("RABBITMQ_URL"))
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	rabbitCh, err = conn.Channel()
	failOnError(err, "RabbitMQ channel failed")
	defer rabbitCh.Close()

	err = rabbitCh.ExchangeDeclare("hospital", "topic", true, false, false, false, nil)
	failOnError(err, "Exchange declare failed")

	// Router
	r := gin.Default()
	r.POST("/appointments", create)
	r.GET("/appointments", list)
	r.PUT("/appointments/:id/confirm", confirm)
	r.PUT("/appointments/:id/cancel", cancel)

	log.Println("Appointment Service running on :8080")
	r.Run(":8080")
}
