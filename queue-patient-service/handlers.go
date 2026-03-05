package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// list แสดงรายการคิว
func list(c *gin.Context) {
	var queues []Queue
	userRole := c.GetString("role")
	userID := c.GetString("user_id")

	query := db.Model(&Queue{})

	// [Logic] ถ้าเป็นคนไข้ ให้เห็นเฉพาะคิวของตัวเองเท่านั้น
	// ถ้าเป็นหมอหรือเจ้าหน้าที่ ให้เห็นคิวทั้งหมด
	if userRole == "patient" {
		query = query.Where("patient_id = ?", userID)
	}

	if err := query.Find(&queues).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch queues"})
		return
	}

	c.JSON(http.StatusOK, queues)
}

// call สำหรับเรียกคิว (เปลี่ยนสถานะจาก waiting -> serving)
func call(c *gin.Context) {
	userRole := c.GetString("role")

	// [Logic] เฉพาะหมอหรือเจ้าหน้าที่เท่านั้นที่เรียกคิวได้
	if userRole != "doctor" && userRole != "staff" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: Only doctors or staff can call patients"})
		return
	}

	id := c.Param("id")
	var q Queue

	if err := db.First(&q, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "queue not found"})
		return
	}

	if q.Status != StatusWaiting {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("cannot call queue with status '%s'", q.Status)})
		return
	}

	q.Status = StatusServing
	if err := db.Save(&q).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "patient called", "queue_number": q.QueueNumber})
}

// done สำหรับปิดคิว (เปลี่ยนสถานะจาก serving -> done)
func done(c *gin.Context) {
	userRole := c.GetString("role")

	// [Logic] เฉพาะหมอหรือเจ้าหน้าที่เท่านั้นที่ปิดคิวได้
	if userRole != "doctor" && userRole != "staff" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: Only doctors or staff can complete treatment"})
		return
	}

	id := c.Param("id")
	var q Queue

	if err := db.First(&q, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "queue not found"})
		return
	}

	if q.Status != StatusServing {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("cannot complete queue with status '%s'", q.Status)})
		return
	}

	q.Status = StatusDone
	if err := db.Save(&q).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update status"})
		return
	}

	// ส่ง Event ไปยัง Payment Service
	publishEvent("queue.done", map[string]interface{}{
		"queue_id":       q.ID,
		"appointment_id": q.AppointmentID,
		"patient_id":     q.PatientID,
	})

	c.JSON(http.StatusOK, gin.H{"message": "treatment done and sent to payment system"})
}
