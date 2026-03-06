package main

import (
	"errors"

	"github.com/KKP240/Q-Hospital/auth/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

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

func getAllPayments(c *gin.Context) {
	var payment []Payment

	if err := db.Find(&payment).Error; err != nil {
		c.JSON(500, gin.H{"error": "Database error"})
		return
	}

	if len(payment) == 0 {
		c.JSON(404, gin.H{"error": "No payments found"})
		return
	}

	c.JSON(200, payment)
}

func getMyPayments(c *gin.Context) {
	currentUserID, _, exists := middleware.GetCurrentUser(c)
	if !exists {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	var payment []Payment
	if err := db.Where("patient_id = ?", currentUserID).Find(&payment).Error; err != nil {
		c.JSON(500, gin.H{"error": "Database error"})
		return
	}

	c.JSON(200, payment)
}

func getPayment(c *gin.Context) {
	currentUserID, role, exists := middleware.GetCurrentUser(c)
	if !exists {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	var payment Payment

	if role == "doctor" {
		if err := db.Where("queue_id = ?", c.Param("queue_id")).First(&payment).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(404, gin.H{"error": "Payment not found"})
				return
			}
			c.JSON(500, gin.H{"error": "Database error"})
			return
		}
	} else {
		if err := db.Where("queue_id = ? AND patient_id = ?", c.Param("queue_id"), currentUserID).First(&payment).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(404, gin.H{"error": "Payment not found"})
				return
			}
			c.JSON(500, gin.H{"error": "Database error"})
			return
		}
	}

	c.JSON(200, payment)
}

func pay(c *gin.Context) {
	id := c.Param("id")

	currentUserID, _, exists := middleware.GetCurrentUser(c)
	if !exists {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

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

		pres := Prescription{QueueID: payment.QueueID, PatientID: currentUserID, Medicine: "Paracetamol, Vit-C"}

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

func getAllPrescriptions(c *gin.Context) {
	var pres []Prescription

	if err := db.Find(&pres).Error; err != nil {
		c.JSON(500, gin.H{
			"error":   "Database error",
			"details": err.Error(),
		})
		return
	}

	if len(pres) == 0 {
		c.JSON(404, gin.H{"error": "No prescriptions found"})
		return
	}

	c.JSON(200, pres)
}

func getMyPrescriptions(c *gin.Context) {
	currentUserID, role, exists := middleware.GetCurrentUser(c)
	if !exists {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	if role != "patient" {
		c.JSON(403, gin.H{"error": "Only patients can access this endpoint"})
		return
	}

	var pres []Prescription
	if err := db.Where("patient_id = ?", currentUserID).Find(&pres).Error; err != nil {
		c.JSON(500, gin.H{"error": "Database error", "details": err.Error()})
		return
	}

	c.JSON(200, pres)
}

func getPrescription(c *gin.Context) {
	currentUserID, role, exists := middleware.GetCurrentUser(c)
	if !exists {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	var pres Prescription

	if role == "doctor" {
		if err := db.Where("queue_id = ?", c.Param("queue_id")).First(&pres).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(404, gin.H{"error": "Prescription not found"})
				return
			}
			c.JSON(500, gin.H{"error": "Database error", "details": err.Error()})
			return
		}
	} else {
		if err := db.Where("queue_id = ? AND patient_id = ?", c.Param("queue_id"), currentUserID).First(&pres).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(404, gin.H{"error": "Prescription not found"})
				return
			}
			c.JSON(500, gin.H{"error": "Database error", "details": err.Error()})
			return
		}
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
