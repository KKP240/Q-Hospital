package main

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Endpoints

func create(c *gin.Context) {
	ctx := c.Request.Context()

	var a Appointment

	if err := c.ShouldBindJSON(&a); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check user role
	userRole := c.GetString("role")

	if userRole != "doctor" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: Only doctors can create appointments"})
		return
	}

	// Verify patient or doctor
	patient, err := GetUser(ctx, a.PatientID)
	if err != nil || patient == nil || patient.Role != "patient" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Patient not found or service unavailable"})
		return
	}

	doctor, err := GetUser(ctx, a.DoctorID)
	if err != nil || doctor == nil || doctor.Role != "doctor" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Doctor not found or service unavailable"})
		return
	}

	// Create Appointment
	if err := db.WithContext(ctx).Create(&a).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create appointment"})
		return
	}

	c.JSON(http.StatusCreated, a)
}

func list(c *gin.Context) {
	ctx := c.Request.Context()
	var apps []Appointment

	// List all appointment
	if err := db.WithContext(ctx).Find(&apps).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch appointments"})
		return
	}

	c.JSON(http.StatusOK, apps)
}

func confirm(c *gin.Context) {
	ctx := c.Request.Context()
	var a Appointment

	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	// Check user role
	userRole := c.GetString("role")

	if userRole != "patient" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: Only patients can modify appointments"})
		return
	}

	if err := db.WithContext(ctx).First(&a, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "appointment not found"})
		return
	}

	if a.Status == "confirmed" || a.Status == "cancelled" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "appointment already " + a.Status})
		return
	}

	// Database Transaction
	tx := db.WithContext(ctx).Begin()
	a.Status = "confirmed"

	if err := tx.Save(&a).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update status"})
		return
	}

	if err := publishEvent(ctx, "appointment.confirmed", a); err != nil {
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
	ctx := c.Request.Context()
	var a Appointment

	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	// Check user role
	userRole := c.GetString("role")

	if userRole != "patient" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: Only patients can modify appointments"})
		return
	}

	if err := db.WithContext(ctx).First(&a, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "appointment not found"})
		return
	}

	if a.Status == "confirmed" || a.Status == "cancelled" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "already " + a.Status})
		return
	}

	a.Status = "cancelled"
	if err := db.WithContext(ctx).Save(&a).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "cancelled successfully"})
}
