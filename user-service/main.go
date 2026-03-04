package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
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
var jwtSecret []byte

func main() {
	godotenv.Load()

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("JWT_SECRET not set")
	}
	jwtSecret = []byte(secret)

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

	r := gin.Default()

	r.POST("/register", register)
	r.POST("/login", login)
	r.GET("/users", getAllUsers)
	r.GET("/users/:id", getUserByID)
	authorized := r.Group("/")
	authorized.Use(AuthMiddleware())
	{
		authorized.GET("/me", getCurrentUser)
	}

	log.Println("User Service running on :8080")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	r.Run(":" + port)

	log.Println("JWT_SECRET =", os.Getenv("JWT_SECRET"))
	log.Println("DB_URL =", os.Getenv("DB_URL"))
	log.Println("RABBITMQ_URL =", os.Getenv("RABBITMQ_URL"))
	log.Println("PORT =", os.Getenv("PORT"))
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

func login(c *gin.Context) {

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

	token, err := generateToken(user)
	if err != nil {
		c.JSON(500, gin.H{"error": "Token generation failed"})
		return
	}

	c.JSON(200, gin.H{
		"message": "Login successful",
		"token":   token,
	})
}

//////////////////// TOKEN //////////////////////

func generateToken(user User) (string, error) {

	claims := jwt.MapClaims{
		"user_id": user.ID,
		"role":    user.Role,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(jwtSecret)
}

//////////////////// AUTH MIDDLEWARE //////////////////////

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenString := strings.SplitN(authHeader, " ", 2)

		if len(tokenString) != 2 || tokenString[0] != "Bearer" {
			c.JSON(401, gin.H{"error": "Invalid authorization format"})
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenString[1], func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.JSON(401, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || !token.Valid {
			c.JSON(401, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}

		c.Set("user_id", claims["user_id"])
		c.Set("role", claims["role"])

		c.Next()
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
