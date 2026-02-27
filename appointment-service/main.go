// appointment-service/main.go
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

type Appointment struct {
	ID      uint   `json:"id" gorm:"primaryKey"`
	Patient string `json:"patient" binding:"required"`
	Doctor  string `json:"doctor" binding:"required"`
	Date    string `json:"date" binding:"required"`
	Status  string `json:"status" gorm:"default:'pending'"`
}

var db *gorm.DB
var rabbitCh *amqp.Channel

func main() {
	var err error
	db, _ = gorm.Open(postgres.Open(os.Getenv("DB_URL")), &gorm.Config{})
	db.AutoMigrate(&Appointment{})

	conn, _ := amqp.Dial(os.Getenv("RABBITMQ_URL"))
	rabbitCh, _ = conn.Channel()
	rabbitCh.ExchangeDeclare("hospital", "topic", true, false, false, false, nil)

	r := gin.Default()
	r.POST("/appointments", create)
	r.GET("/appointments", list)
	r.PUT("/appointments/:id/confirm", confirm) // ส่งไป queue-patient
	r.PUT("/appointments/:id/cancel", cancel)

	log.Println("Appointment Service :8080")
	r.Run(":8080")
}

func create(c *gin.Context) {
	var a Appointment
	c.ShouldBindJSON(&a)
	db.Create(&a)
	c.JSON(201, a)
}

func list(c *gin.Context) {
	var apps []Appointment
	db.Find(&apps)
	c.JSON(200, apps)
}

func confirm(c *gin.Context) {
	id := c.Param("id")
	var a Appointment
	db.First(&a, id)
	a.Status = "confirmed"
	db.Save(&a)

	// ส่งไป queue-patient service
	event, _ := json.Marshal(map[string]interface{}{
		"appointment_id": a.ID,
		"patient":        a.Patient,
		"doctor":         a.Doctor,
	})
	rabbitCh.Publish("hospital", "appointment.confirmed", false, false, amqp.Publishing{
		ContentType: "application/json", Body: event,
	})

	c.JSON(200, gin.H{"message": "confirmed, sent to queue", "data": a})
}

func cancel(c *gin.Context) {
	db.Model(&Appointment{}).Where("id = ?", c.Param("id")).Update("status", "cancelled")
	c.JSON(200, gin.H{"message": "cancelled"})
}
