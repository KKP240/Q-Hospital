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

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET not set")
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

	// middleware
	authConfig := auth.NewAuthConfig(jwtSecret)

	authorized := r.Group("/")
	authorized.Use(middleware.GinAuthMiddleware(authConfig))
	{
		authorized.GET("/appointments", list)
		authorized.POST("/appointments", create)
		authorized.PUT("/appointments/:id/confirm", confirm)
		authorized.PUT("/appointments/:id/cancel", cancel)
	}

	log.Println("Appointment Service running on :8080")
	r.Run(":8080")
}
