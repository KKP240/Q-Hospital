package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/streadway/amqp"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	StatusWaiting = "waiting"
	StatusServing = "serving"
	StatusDone    = "done"
	StatusSkipped = "skipped"
)

type Queue struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	AppointmentID uint      `json:"appointment_id"`
	Patient       string    `json:"patient"`
	Doctor        string    `json:"doctor"`
	QueueNumber   string    `json:"queue_number" gorm:"uniqueIndex"`
	Status        string    `json:"status" gorm:"default:'waiting'"`
	CreatedAt     time.Time `json:"created_at"`
}

// 🔥 GORM Hook: Auto-generate QueueNumber หลัง create
func (q *Queue) AfterCreate(tx *gorm.DB) (err error) {
	q.QueueNumber = fmt.Sprintf("A%03d", q.ID)
	return tx.Model(q).Update("queue_number", q.QueueNumber).Error
}

var db *gorm.DB
var rabbitCh *amqp.Channel

func main() {
	var err error

	// ================= DB =================
	db, err = gorm.Open(postgres.Open(os.Getenv("DB_URL")), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect database:", err)
	}

	if err := db.AutoMigrate(&Queue{}); err != nil {
		log.Fatal("Migration failed:", err)
	}

	// ================= RabbitMQ =================
	conn, err := amqp.Dial(os.Getenv("RABBITMQ_URL"))
	if err != nil {
		log.Fatal("Failed to connect RabbitMQ:", err)
	}

	rabbitCh, err = conn.Channel()
	if err != nil {
		log.Fatal("Failed to open channel:", err)
	}

	if err := rabbitCh.ExchangeDeclare("hospital", "topic", true, false, false, false, nil); err != nil {
		log.Fatal("Exchange declare failed:", err)
	}

	q, err := rabbitCh.QueueDeclare("queue-patient", true, false, false, false, nil)
	if err != nil {
		log.Fatal("Queue declare failed:", err)
	}

	if err := rabbitCh.QueueBind(q.Name, "appointment.confirmed", "hospital", false, nil); err != nil {
		log.Fatal("Queue bind failed:", err)
	}

	// 🔥 Manual ACK
	msgs, err := rabbitCh.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		log.Fatal("Consume failed:", err)
	}

	go func() {
		for msg := range msgs {

			var event map[string]interface{}
			if err := json.Unmarshal(msg.Body, &event); err != nil {
				log.Println("Invalid message format:", err)
				msg.Nack(false, false)
				continue
			}

			appointmentID, ok := event["appointment_id"].(float64)
			if !ok {
				log.Println("Invalid appointment_id")
				msg.Nack(false, false)
				continue
			}

			queue := Queue{
				AppointmentID: uint(appointmentID),
				Patient:       event["patient"].(string),
				Doctor:        event["doctor"].(string),
				Status:        StatusWaiting,
			}

			if err := db.Create(&queue).Error; err != nil {
				log.Println("Failed to create queue:", err)
				msg.Nack(false, true)
				continue
			}

			log.Printf("Created queue %s", queue.QueueNumber)
			msg.Ack(false)
		}
	}()

	// ================= API =================
	r := gin.Default()

	r.GET("/queues", list)
	r.PUT("/queues/:id/call", call)
	r.PUT("/queues/:id/done", done)

	log.Println("Queue-Patient Service running on :8080")
	r.Run(":8080")
}

// ================= Handlers =================

func list(c *gin.Context) {
	var queues []Queue
	if err := db.Find(&queues).Error; err != nil {
		c.JSON(500, gin.H{"error": "failed to fetch queues"})
		return
	}
	c.JSON(200, queues)
}

func call(c *gin.Context) {
	id := c.Param("id")

	var q Queue
	if err := db.First(&q, id).Error; err != nil {
		c.JSON(404, gin.H{"error": "queue not found"})
		return
	}

	if q.Status != StatusWaiting {
		c.JSON(400, gin.H{
			"error": fmt.Sprintf("cannot call queue with status '%s'", q.Status),
		})
		return
	}

	q.Status = StatusServing
	if err := db.Save(&q).Error; err != nil {
		c.JSON(500, gin.H{"error": "failed to update status"})
		return
	}

	c.JSON(200, gin.H{"message": "patient called"})
}

func done(c *gin.Context) {
	id := c.Param("id")

	var q Queue
	if err := db.First(&q, id).Error; err != nil {
		c.JSON(404, gin.H{"error": "queue not found"})
		return
	}

	if q.Status != StatusServing {
		c.JSON(400, gin.H{
			"error": fmt.Sprintf("cannot complete queue with status '%s'", q.Status),
		})
		return
	}

	q.Status = StatusDone
	if err := db.Save(&q).Error; err != nil {
		c.JSON(500, gin.H{"error": "failed to update status"})
		return
	}

	// Publish event
	event, err := json.Marshal(map[string]interface{}{
		"queue_id": q.ID,
		"patient":  q.Patient,
	})
	if err == nil {
		rabbitCh.Publish("hospital", "queue.done", false, false, amqp.Publishing{
			ContentType: "application/json",
			Body:        event,
		})
	}

	c.JSON(200, gin.H{"message": "done, sent to payment"})
}
