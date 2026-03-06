package main

type Payment struct {
	ID        uint    `json:"id" gorm:"primaryKey"`
	QueueID   uint    `json:"queue_id"`
	PatientID string  `json:"patient_id"`
	Amount    float64 `json:"amount"`
	Status    string  `json:"status" gorm:"default:'pending'"`
}

type Prescription struct {
	ID        uint   `json:"id" gorm:"primaryKey"`
	QueueID   uint   `json:"queue_id"`
	PatientID string `json:"patient_id"`
	Medicine  string `json:"medicine"`
	Status    string `json:"status" gorm:"default:'pending'"`
}
