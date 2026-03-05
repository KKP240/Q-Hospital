package main

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Endpoints

func create(c *gin.Context) {
	var a Appointment

	if err := c.ShouldBindJSON(&a); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify patient or doctor
	patient, err := GetPatient(a.PatientID)
	if err != nil || patient == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Patient not found or service unavailable"})
		return
	}

	doctor, err := GetDoctor(a.DoctorID)
	if err != nil || doctor == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Doctor not found or service unavailable"})
		return
	}

	// Create Appointment
	if err := db.Create(&a).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create appointment"})
		return
	}

	c.JSON(http.StatusCreated, a)
}

func list(c *gin.Context) {
	var apps []Appointment

	// List all appointment
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

	if a.Status == "confirmed" || a.Status == "cancelled" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "appointment already " + a.Status})
		return
	}

	// Database Transaction
	tx := db.Begin()
	a.Status = "confirmed"

	if err := tx.Save(&a).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update status"})
		return
	}

	if err := publishEvent("appointment.confirmed", a); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to publish event, status reverted"})
		return
	}

	tx.Commit()

	c.JSON(http.StatusOK, gin.H{
		"message": "confirmed and event published",
		"data":    a,
	})
}

func cancel(c *gin.Context) {
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

	if a.Status == "confirmed" || a.Status == "cancelled" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "already " + a.Status})
		return
	}

	a.Status = "cancelled"
	if err := db.Save(&a).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "cancelled successfully"})
}
