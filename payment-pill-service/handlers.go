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

func getPayment(c *gin.Context) {
	var payment Payment
	if err := db.Where("queue_id = ?", c.Param("queue_id")).First(&payment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(404, gin.H{"error": "Payment not found"})
			return
		}
		c.JSON(500, gin.H{"error": "Database error"})
		return
	}
	c.JSON(200, payment)
}

func pay(c *gin.Context) {
	id := c.Param("id")

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

		// สร้างใบสั่งยา
		pres := Prescription{QueueID: payment.QueueID, Medicine: "Paracetamol, Vit-C"}

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

func getPrescription(c *gin.Context) {
	var pres Prescription
	if err := db.Where("queue_id = ?", c.Param("queue_id")).First(&pres).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(404, gin.H{"error": "Prescription not found"})
			return
		}
		c.JSON(500, gin.H{"error": "Database error", "details": err.Error()})
		return
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
