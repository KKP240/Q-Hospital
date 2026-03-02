package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"

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

// ============ Render Error ============
func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func main() {
	var err error

	// ============ DB ============
	db, err = gorm.Open(postgres.Open(os.Getenv("DB_URL")), &gorm.Config{})
	failOnError(err, "Failed to connect to Database.")

	err = db.AutoMigrate(&Appointment{})
	failOnError(err, "Migration failed.")

	// ============ RabbitMQ ============
	conn, err := amqp.Dial(os.Getenv("RABBITMQ_URL"))
	failOnError(err, "Failed to connect to RabbitMQ.")

	defer conn.Close()

	rabbitCh, err = conn.Channel()
	failOnError(err, "RabbitMQ channel failed.")

	defer rabbitCh.Close()

	err = rabbitCh.ExchangeDeclare("hospital", "topic", true, false, false, false, nil)
	failOnError(err, "Exchange declare failed.")

	// ============ Router ============
	r := gin.Default()

	r.POST("/appointments", create)
	r.GET("/appointments", list)
	r.PUT("/appointments/:id/confirm", confirm)
	r.PUT("/appointments/:id/cancel", cancel)

	log.Println("Appointment Service :8080")
	r.Run(":8080")
}

// ============ Endpoints ============

func create(c *gin.Context) {
	var a Appointment

	if err := c.ShouldBindJSON(&a); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := db.Create(&a).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create appointment"})
		return
	}

	c.JSON(http.StatusCreated, a)
}

func list(c *gin.Context) {
	var apps []Appointment

	if err := db.Find(&apps).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch appointments"})
		return
	}

	c.JSON(http.StatusOK, apps)
}

func confirm(c *gin.Context) {
	var a Appointment

	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := db.First(&a, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "appointment not found"})
		return
	}

	if a.Status == "confirmed" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "already confirmed"})
		return
	}

	a.Status = "confirmed"

	if err := db.Save(&a).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update status"})
		return
	}

	if err := publishEvent("appointment.confirmed", a); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to publish event"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "confirmed and event published",
		"data":    a,
	})
}

func cancel(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	result := db.Model(&Appointment{}).
		Where("id = ? AND status != ?", id, "cancelled").
		Update("status", "cancelled")

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "appointment not found"})
		return
	}

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "cancelled"})
}

// ============ RabitMQ Publisher ============
func publishEvent(routingKey string, a Appointment) error {
	event, err := json.Marshal(map[string]interface{}{
		"appointment_id": a.ID,
		"patient":        a.Patient,
		"doctor":         a.Doctor,
	})

	if err != nil {
		return err
	}

	return rabbitCh.Publish(
		"hospital",
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        event,
		},
	)
}
