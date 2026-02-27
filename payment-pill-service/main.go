// payment-pill-service/main.go
package main

import (
	"encoding/json"
	"log"
	"os"

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
	// var err error
	db, _ = gorm.Open(postgres.Open(os.Getenv("DB_URL")), &gorm.Config{})
	db.AutoMigrate(&Payment{}, &Prescription{})

	conn, _ := amqp.Dial(os.Getenv("RABBITMQ_URL"))
	ch, _ := conn.Channel()
	ch.ExchangeDeclare("hospital", "topic", true, false, false, false, nil)

	// รับจาก queue-patient
	q, _ := ch.QueueDeclare("payment-pill", true, false, false, false, nil)
	ch.QueueBind(q.Name, "queue.done", "hospital", false, nil)

	go func() {
		msgs, _ := ch.Consume(q.Name, "", true, false, false, false, nil)
		for msg := range msgs {
			var event map[string]interface{}
			json.Unmarshal(msg.Body, &event)
			queueID := uint(event["queue_id"].(float64))

			// สร้างใบเสร็จอัตโนมัติ
			payment := Payment{QueueID: queueID, Amount: 500}
			db.Create(&payment)
			log.Printf("Created payment for queue %d", queueID)
		}
	}()

	r := gin.Default()

	// Payment APIs
	r.GET("/payments/:queue_id", getPayment)
	r.PUT("/payments/:id/pay", pay) // จ่ายเงิน → สร้างใบสั่งยา

	// Prescription APIs
	r.GET("/prescriptions/:queue_id", getPrescription)
	r.PUT("/prescriptions/:id/dispense", dispense)

	log.Println("Payment-Pill Service :8080")
	r.Run(":8080")
}

func getPayment(c *gin.Context) {
	var p Payment
	db.Where("queue_id = ?", c.Param("queue_id")).First(&p)
	c.JSON(200, p)
}

func pay(c *gin.Context) {
	id := c.Param("id")
	db.Model(&Payment{}).Where("id = ?", id).Update("status", "paid")

	// สร้างใบสั่งยาอัตโนมัติ
	var pay Payment
	db.First(&pay, id)
	pres := Prescription{QueueID: pay.QueueID, Medicine: "Paracetamol, Vit-C"}
	db.Create(&pres)

	c.JSON(200, gin.H{"message": "paid, prescription created", "prescription": pres})
}

func getPrescription(c *gin.Context) {
	var p Prescription
	db.Where("queue_id = ?", c.Param("queue_id")).First(&p)
	c.JSON(200, p)
}

func dispense(c *gin.Context) {
	db.Model(&Prescription{}).Where("id = ?", c.Param("id")).Update("status", "dispensed")
	c.JSON(200, gin.H{"message": "medicine dispensed"})
}
