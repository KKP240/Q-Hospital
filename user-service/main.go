package main

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/KKP240/Q-Hospital/auth"
	"github.com/KKP240/Q-Hospital/auth/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/streadway/amqp"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type User struct {
	ID        string    `json:"id" gorm:"type:uuid;primaryKey"`
	Name      string    `json:"name" gorm:"not null"`
	Email     string    `json:"email" gorm:"unique;not null"`
	Password  string    `json:"-" gorm:"not null"`
	Role      string    `json:"role" gorm:"type:varchar(20);not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Patient struct {
	UserID      string `json:"user_id" gorm:"type:uuid;primaryKey"`
	User        User   `json:"user" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Address     string `json:"address"`
	PhoneNumber string `json:"phone_number"`
}

type Doctor struct {
	UserID      string `json:"user_id" gorm:"type:uuid;primaryKey"`
	User        User   `json:"user" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Address     string `json:"address"`
	PhoneNumber string `json:"phone_number"`
}

var db *gorm.DB
var rabbitChannel *amqp.Channel

func main() {
	godotenv.Load()

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET not set")
	}

	var err error
	db, err = gorm.Open(postgres.Open(os.Getenv("DB_URL")), &gorm.Config{})
	if err != nil {
		log.Fatalf("DB connection failed: %v", err)
	}

	db.AutoMigrate(&User{}, &Patient{}, &Doctor{})

	conn, err := amqp.Dial(os.Getenv("RABBITMQ_URL"))
	if err != nil {
		log.Fatalf("RabbitMQ failed: %v", err)
	}

	rabbitChannel, _ = conn.Channel()
	rabbitChannel.ExchangeDeclare("hospital", "topic", true, false, false, false, nil)

	authConfig := auth.NewAuthConfig(jwtSecret)

	r := gin.Default()

	r.POST("/register", register)
	r.POST("/login", login(authConfig)) // ส่ง config เข้าไป
	r.GET("/users", getAllUsers)
	r.GET("/users/:id", getUserByID)
	r.GET("/patients/:id", getPatientByID)
	r.GET("/doctors/:id", getDoctorByID)

	// ใช้ shared middleware
	authorized := r.Group("/")
	authorized.Use(middleware.GinAuthMiddleware(authConfig))
	{
		authorized.GET("/me", getCurrentUser)
	}

	log.Println("User Service running on :8080")
	r.Run(":8080")
}

//////////////////// REGISTER //////////////////////

type RegisterInput struct {
	Name        string `json:"name" binding:"required"`
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required"`
	Role        string `json:"role" binding:"required,oneof=patient doctor"`
	Address     string `json:"address"`
	PhoneNumber string `json:"phone_number"`
}

func register(c *gin.Context) {

	var input RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(500, gin.H{"error": "Password hashing failed"})
		return
	}

	userID := uuid.New().String()

	err = db.Transaction(func(tx *gorm.DB) error {

		user := User{
			ID:       userID,
			Name:     input.Name,
			Email:    input.Email,
			Password: string(hashedPassword),
			Role:     input.Role,
		}

		if err := tx.Create(&user).Error; err != nil {
			return err
		}

		if input.Role == "patient" {
			patient := Patient{
				UserID:      userID,
				Address:     input.Address,
				PhoneNumber: input.PhoneNumber,
			}
			if err := tx.Create(&patient).Error; err != nil {
				return err
			}
		}

		if input.Role == "doctor" {
			doctor := Doctor{
				UserID:      userID,
				Address:     input.Address,
				PhoneNumber: input.PhoneNumber,
			}
			if err := tx.Create(&doctor).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// Publish event
	event := map[string]interface{}{
		"event": "user.created",
		"id":    userID,
		"role":  input.Role,
	}

	body, _ := json.Marshal(event)

	rabbitChannel.Publish(
		"hospital",
		"user.created",
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)

	c.JSON(201, gin.H{
		"id":    userID,
		"email": input.Email,
		"role":  input.Role,
	})
}

//////////////////// LOGIN /////////////////////////

type LoginInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func login(authConfig *auth.AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input LoginInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		var user User
		if err := db.Where("email = ?", input.Email).First(&user).Error; err != nil {
			c.JSON(401, gin.H{"error": "Invalid email or password"})
			return
		}

		err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password))
		if err != nil {
			c.JSON(401, gin.H{"error": "Invalid email or password"})
			return
		}

		token, err := authConfig.GenerateToken(user.ID, user.Role, time.Hour*24)
		if err != nil {
			c.JSON(500, gin.H{"error": "Token generation failed"})
			return
		}

		c.JSON(200, gin.H{
			"message": "Login successful",
			"token":   token,
		})
	}
}

//////////////////// GET CURRENT USER //////////////////////

func getCurrentUser(c *gin.Context) {

	userID := c.GetString("user_id")

	var user User
	if err := db.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(404, gin.H{"error": "User not found"})
		return
	}

	c.JSON(200, gin.H{
		"id":    user.ID,
		"name":  user.Name,
		"email": user.Email,
		"role":  user.Role,
	})
}

//////////////////// GET USER //////////////////////

func getAllUsers(c *gin.Context) {

	var users []User
	if err := db.Find(&users).Error; err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	var result []gin.H

	for _, user := range users {

		response := gin.H{
			"id":    user.ID,
			"name":  user.Name,
			"email": user.Email,
			"role":  user.Role,
		}

		if user.Role == "patient" {
			var patient Patient
			db.First(&patient, "user_id = ?", user.ID)

			response["profile"] = gin.H{
				"address":      patient.Address,
				"phone_number": patient.PhoneNumber,
			}
		}

		if user.Role == "doctor" {
			var doctor Doctor
			db.First(&doctor, "user_id = ?", user.ID)

			response["profile"] = gin.H{
				"address":      doctor.Address,
				"phone_number": doctor.PhoneNumber,
			}
		}

		result = append(result, response)
	}

	c.JSON(200, result)
}

func getUserByID(c *gin.Context) {

	id := c.Param("id")

	var user User
	if err := db.First(&user, "id = ?", id).Error; err != nil {
		c.JSON(404, gin.H{"error": "User not found"})
		return
	}

	response := gin.H{
		"id":         user.ID,
		"name":       user.Name,
		"email":      user.Email,
		"role":       user.Role,
		"created_at": user.CreatedAt,
		"updated_at": user.UpdatedAt,
	}

	if user.Role == "patient" {
		var patient Patient
		db.First(&patient, "user_id = ?", user.ID)

		response["profile"] = gin.H{
			"address":      patient.Address,
			"phone_number": patient.PhoneNumber,
		}
	}

	if user.Role == "doctor" {
		var doctor Doctor
		db.First(&doctor, "user_id = ?", user.ID)

		response["profile"] = gin.H{
			"address":      doctor.Address,
			"phone_number": doctor.PhoneNumber,
		}
	}

	c.JSON(200, response)
}

func getPatientByID(c *gin.Context) {
	id := c.Param("id")

	var patient Patient
	if err := db.First(&patient, "user_id = ?", id).Error; err != nil {
		c.JSON(404, gin.H{"error": "Patient not found"})
		return
	}

	response := gin.H{
		"user_id":      patient.UserID,
		"address":      patient.Address,
		"phone_number": patient.PhoneNumber,
		"user":         patient.User,
	}

	c.JSON(200, response)
}

func getDoctorByID(c *gin.Context) {
	id := c.Param("id")

	var doctor Doctor
	if err := db.First(&doctor, "user_id = ?", id).Error; err != nil {
		c.JSON(404, gin.H{"error": "Doctor not found"})
		return
	}

	response := gin.H{
		"user_id":      doctor.UserID,
		"address":      doctor.Address,
		"phone_number": doctor.PhoneNumber,
		"user":         doctor.User,
	}

	c.JSON(200, response)
}
