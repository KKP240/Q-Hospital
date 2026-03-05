package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func list(c *gin.Context) {

	var queues []Queue

	if err := db.Find(&queues).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch queues"})
		return
	}

	c.JSON(http.StatusOK, queues)
}

func call(c *gin.Context) {

	id := c.Param("id")

	var q Queue

	if err := db.First(&q, id).Error; err != nil {
		c.JSON(404, gin.H{"error": "queue not found"})
		return
	}

	if q.Status != StatusWaiting {
		c.JSON(400, gin.H{"error": fmt.Sprintf("cannot call queue with status '%s'", q.Status)})
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
		c.JSON(400, gin.H{"error": fmt.Sprintf("cannot complete queue with status '%s'", q.Status)})
		return
	}

	q.Status = StatusDone

	if err := db.Save(&q).Error; err != nil {
		c.JSON(500, gin.H{"error": "failed to update status"})
		return
	}

	publishEvent("queue.done", map[string]interface{}{
		"queue_id": q.ID,
		"patient":  q.PatientID,
	})

	c.JSON(200, gin.H{"message": "done and sent to payment"})
}
