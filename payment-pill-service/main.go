package main

import (
	"log"
	"os"

	"github.com/KKP240/Q-Hospital/auth"
	"github.com/KKP240/Q-Hospital/auth/middleware"

	"github.com/gin-gonic/gin"
	"github.com/streadway/amqp"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

func main() {
	// connection handling
	var err error
	db, err = gorm.Open(postgres.Open(os.Getenv("DB_URL")), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	db.AutoMigrate(&Payment{}, &Prescription{})

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET not set")
	}
	authConfig := auth.NewAuthConfig(jwtSecret)

	conn, err := amqp.Dial(os.Getenv("RABBITMQ_URL"))
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to opan a channel: %v", err)
	}
	defer ch.Close()

	initRabbitMQConsumer(ch, db)

	r := gin.Default()

	authorized := r.Group("/")
	authorized.Use(middleware.GinAuthMiddleware(authConfig))
	{
		authorized.GET("/me", getCurrentUser)

		pharmacist := authorized.Group("/")
		pharmacist.Use(middleware.RequireRole("pharmacist", "admin", "doctor"))
		{
			pharmacist.PUT("/prescriptions/:id/dispense", dispense)
		}

		cashier := authorized.Group("/")
		cashier.Use(middleware.RequireRole("cashier", "admin", "patient"))
		{
			cashier.PUT("/payments/:id/pay", pay)
		}

		authorized.GET("/payments/:queue_id", getPayment)
		authorized.GET("/prescriptions/:queue_id", getPrescription)
	}

	log.Println("Payment-Pill Service :8080")
	r.Run(":8080")
}
