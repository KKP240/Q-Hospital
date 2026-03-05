package main

type Appointment struct {
	ID        uint   `json:"id" gorm:"primaryKey"`
	PatientID string `json:"patient_id" binding:"required"`
	DoctorID  string `json:"doctor_id" binding:"required"`
	Date      string `json:"date" binding:"required"`
	Status    string `json:"status" gorm:"default:'pending'"`
}

type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type Patient struct {
	UserID      string `json:"user_id"`
	Address     string `json:"address"`
	PhoneNumber string `json:"phone_number"`
	User        User   `json:"user"`
}

type Doctor struct {
	UserID      string `json:"user_id"`
	Address     string `json:"address"`
	PhoneNumber string `json:"phone_number"`
	User        User   `json:"user"`
}
