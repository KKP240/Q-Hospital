package middleware

import (
	"net/http"
	"strings"

	"github.com/KKP240/Q-Hospital/auth"
	"github.com/gin-gonic/gin"
)

// GinAuthMiddleware สร้าง Gin middleware สำหรับตรวจสอบ JWT
func GinAuthMiddleware(config *auth.AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		tokenString, err := auth.ExtractTokenFromHeader(authHeader)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		claims, err := config.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Set user info in context ให้ handler อื่นๆ ใช้ได้
		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)

		c.Next()
	}
}

// RequireRole middleware สำหรับตรวจสอบ role เฉพาะเจาะจง
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "Role not found in context"})
			c.Abort()
			return
		}

		roleStr, ok := userRole.(string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "Invalid role type"})
			c.Abort()
			return
		}

		for _, r := range roles {
			if strings.EqualFold(r, roleStr) {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{
			"error":          "Insufficient permissions",
			"required_roles": roles,
			"your_role":      roleStr,
		})
		c.Abort()
	}
}

// GetCurrentUser ดึงข้อมูล user จาก context
func GetCurrentUser(c *gin.Context) (userID, role string, exists bool) {
	userIDVal, userExists := c.Get("user_id")
	roleVal, roleExists := c.Get("role")

	if !userExists || !roleExists {
		return "", "", false
	}

	// ✅ Type assertion แยกตัวแปร
	userIDStr, ok1 := userIDVal.(string)
	roleStr, ok2 := roleVal.(string)

	if !ok1 || !ok2 {
		return "", "", false
	}

	return userIDStr, roleStr, true
}
