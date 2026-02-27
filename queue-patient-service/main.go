// queue-patient-service/main.go
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

type Queue struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	AppointmentID uint      `json:"appointment_id"`
	Patient       string    `json:"patient"`
	Doctor        string    `json:"doctor"`
	QueueNumber   string    `json:"queue_number"`
	Status        string    `json:"status" gorm:"default:'waiting'"` // waiting, serving, done, skipped
	CreatedAt     time.Time `json:"created_at"`
}

var db *gorm.DB
var rabbitCh *amqp.Channel
var counter = 0

func main() {
	var err error
	db, _ = gorm.Open(postgres.Open(os.Getenv("DB_URL")), &gorm.Config{})
	db.AutoMigrate(&Queue{})

	conn, _ := amqp.Dial(os.Getenv("RABBITMQ_URL"))
	rabbitCh, _ = conn.Channel()
	rabbitCh.ExchangeDeclare("hospital", "topic", true, false, false, false, nil)

	// รับจาก appointment
	q, _ := rabbitCh.QueueDeclare("queue-patient", true, false, false, false, nil)
	rabbitCh.QueueBind(q.Name, "appointment.confirmed", "hospital", false, nil)

	go func() {
		msgs, _ := rabbitCh.Consume(q.Name, "", true, false, false, false, nil)
		for msg := range msgs {
			var event map[string]interface{}
			json.Unmarshal(msg.Body, &event)
			
			counter++
			queue := Queue{
				AppointmentID: uint(event["appointment_id"].(float64)),
				Patient:       event["patient"].(string),
				Doctor:        event["doctor"].(string),
				QueueNumber:   fmt.Sprintf("A%03d", counter),
			}
			db.Create(&queue)
			log.Printf("Created queue %s", queue.QueueNumber)
		}
	}()

	r := gin.Default()
	r.GET("/queues", list)
	r.PUT("/queues/:id/call", call)   // เรียกเข้าห้องตรวจ
	r.PUT("/queues/:id/done", done)   // ตรวจเสร็จ → ส่งไป payment-pill

	log.Println("Queue-Patient Service :8080")
	r.Run(":8080")
}

func list(c *gin.Context) {
	var queues []Queue
	db.Find(&queues)
	c.JSON(200, queues)
}

func call(c *gin.Context) {
	db.Model(&Queue{}).Where("id = ?", c.Param("id")).Update("status", "serving")
	c.JSON(200, gin.H{"message": "patient called"})
}

func done(c *gin.Context) {
	id := c.Param("id")
	var q Queue
	db.First(&q, id)
	q.Status = "done"
	db.Save(&q)

	// ส่งไป payment-pill service
	event, _ := json.Marshal(map[string]interface{}{
		"queue_id": q.ID,
		"patient":  q.Patient,
	})
	rabbitCh.Publish("hospital", "queue.done", false, false, amqp.Publishing{
		ContentType: "application/json", Body: event,
	})

	c.JSON(200, gin.H{"message": "done, sent to payment"})
}