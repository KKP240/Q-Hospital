package main

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

const (
	StatusWaiting = "waiting"
	StatusServing = "serving"
	StatusDone    = "done"
	StatusSkipped = "skipped"
)

type Queue struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	AppointmentID uint      `json:"appointment_id"`
	PatientID     string    `json:"patient_id"`
	DoctorID      string    `json:"doctor_id"`
	QueueNumber   string    `json:"queue_number" gorm:"uniqueIndex"`
	Status        string    `json:"status" gorm:"default:'waiting'"`
	CreatedAt     time.Time `json:"created_at"`
}

// Auto generate QueueNumber
func (q *Queue) AfterCreate(tx *gorm.DB) (err error) {
	q.QueueNumber = fmt.Sprintf("A%03d", q.ID)
	return tx.Model(q).Update("queue_number", q.QueueNumber).Error
}
