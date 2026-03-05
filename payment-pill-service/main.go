package main

import (
	"encoding/json"
	"errors"
	"log"
	"os"

	"github.com/KKP240/Q-Hospital/auth"
	"github.com/KKP240/Q-Hospital/auth/middleware"

	"github.com/gin-gonic/gin"
	"github.com/streadway/amqp"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Payment struct {
	ID      uint    `json:"id" gorm:"primaryKey"`
	QueueID uint    `json:"queue_id"`
	Amount  float64 `json:"amount"`
	Status  string  `json:"status" gorm:"default:'pending'"`
}

type Prescription struct {
	ID       uint   `json:"id" gorm:"primaryKey"`
	QueueID  uint   `json:"queue_id"`
	Medicine string `json:"medicine"`
	Status   string `json:"status" gorm:"default:'pending'"`
}

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

	// รับข้อมูลจาก queue-patient
	err = ch.ExchangeDeclare("hospital", "topic", true, false, false, false, nil)
	q, err := ch.QueueDeclare("payment-pill", true, false, false, false, nil)
	ch.QueueBind(q.Name, "queue.done", "hospital", false, nil)

	go func() {
		msgs, _ := ch.Consume(q.Name, "", false, false, false, false, nil)
		for msg := range msgs {
			var event map[string]interface{}
			if err := json.Unmarshal(msg.Body, &event); err != nil {
				log.Printf("Error decoding message: %v", err)
				continue
			}

			queueID, ok := event["queue_id"].(float64)
			if !ok {
				log.Println("Invalid queue_id in message")
				continue
			}

			var count int64
			db.Model(&Payment{}).Where("queue_id = ?", uint(queueID)).Count(&count)
			if count > 0 {
				msg.Ack(false)
				continue
			}

			// สร้างใบเสร็จ
			payment := Payment{QueueID: uint(queueID), Amount: 500}
			if err := db.Create(&payment).Error; err != nil {
				log.Printf("Failed to create: %v", err)
				msg.Nack(false, true)
			} else {
				msg.Ack(false)
			}
		}
	}()

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
		cashier.Use(middleware.RequireRole("cashier", "admin", "doctor"))
		{
			cashier.PUT("/payments/:id/pay", pay)
		}

		authorized.GET("/payments/:queue_id", getPayment)
		authorized.GET("/prescriptions/:queue_id", getPrescription)
	}

	log.Println("Payment-Pill Service :8080")
	r.Run(":8080")
}

func getCurrentUser(c *gin.Context) {
	userID, role, exists := middleware.GetCurrentUser(c)
	if !exists {
		c.JSON(401, gin.H{"error": "User not found in context"})
		return
	}

	c.JSON(200, gin.H{
		"user_id": userID,
		"role":    role,
		"service": "payment-pill",
	})
}

func getPayment(c *gin.Context) {
	var payment Payment
	if err := db.Where("queue_id = ?", c.Param("queue_id")).First(&payment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(404, gin.H{"error": "Payment not found"})
			return
		}
		c.JSON(500, gin.H{"error": "Database error"})
		return
	}
	c.JSON(200, payment)
}

func pay(c *gin.Context) {
	id := c.Param("id")

	err := db.Transaction(func(ts *gorm.DB) error {
		var payment Payment

		if err := ts.First(&payment, id).Error; err != nil {
			return err
		}

		if payment.Status == "paid" {
			return errors.New("ALREADY_PAID")
		}

		if err := ts.Model(&payment).Update("status", "paid").Error; err != nil {
			return err
		}

		// สร้างใบสั่งยา
		pres := Prescription{QueueID: payment.QueueID, Medicine: "Paracetamol, Vit-C"}

		if err := ts.Create(&pres).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(404, gin.H{"error": "payment record not found"})
			return
		}
		if err.Error() == "ALREADY_PAID" {
			c.JSON(400, gin.H{"error": "this invoice has already been paid"})
			return
		}
		c.JSON(500, gin.H{"error": "Payment failed", "details": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "paid success and prescription created"})
}

func getPrescription(c *gin.Context) {
	var pres Prescription
	if err := db.Where("queue_id = ?", c.Param("queue_id")).First(&pres).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(404, gin.H{"error": "Prescription not found"})
			return
		}
		c.JSON(500, gin.H{"error": "Database error", "details": err.Error()})
		return
	}
	c.JSON(200, pres)
}

func dispense(c *gin.Context) {
	id := c.Param("id")

	err := db.Transaction(func(ts *gorm.DB) error {
		var pres Prescription

		if err := ts.First(&pres, id).Error; err != nil {
			c.JSON(404, gin.H{"error": "Prescription not found"})
			return err
		}

		if pres.Status == "dispensed" {
			return errors.New("ALREADY_DISPENSED")
		}

		var payment Payment
		if err := ts.Where("queue_id = ?", pres.QueueID).First(&payment).Error; err != nil {
			return err
		}

		if payment.Status != "paid" {
			return errors.New("PAYMENT_PENDING")
		}

		if err := ts.Model(&pres).Update("status", "dispensed").Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(404, gin.H{"error": "Record not found"})
			return
		}
		if err.Error() == "ALREADY_DISPENSED" {
			c.JSON(400, gin.H{"error": "This prescription has already been dispensed"})
			return
		}
		if err.Error() == "PAYMENT_PENDING" {
			c.JSON(400, gin.H{"error": "Cannot dispense: Payment is still pending"})
			return
		}
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "medicine dispensed and queue cleared"})
}
