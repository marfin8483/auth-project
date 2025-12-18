package middleware

import (
	"auth-api/config"
	"auth-api/utils"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

func JWTAuth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.ErrorResponse(c, 401, gin.H{"message": "Authorization header required"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			utils.ErrorResponse(c, 401, gin.H{"message": "Invalid authorization format"})
			c.Abort()
			return
		}

		token, err := jwt.Parse(parts[1], func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(cfg.JWT.Secret), nil
		})

		if err != nil {
			utils.ErrorResponse(c, 401, gin.H{"message": "Invalid token", "error": err.Error()})
			c.Abort()
			return
		}

		if !token.Valid {
			utils.ErrorResponse(c, 401, gin.H{"message": "Token is invalid or expired"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			utils.ErrorResponse(c, 401, gin.H{"message": "Invalid token claims"})
			c.Abort()
			return
		}

		// Check token expiration
		exp, ok := claims["exp"].(float64)
		if !ok || time.Now().Unix() > int64(exp) {
			utils.ErrorResponse(c, 401, gin.H{"message": "Token has expired"})
			c.Abort()
			return
		}

		// Extract user_id with proper type handling
		var userID uint
		if uid, ok := claims["user_id"].(float64); ok {
			userID = uint(uid)
		} else {
			utils.ErrorResponse(c, 401, gin.H{"message": "Invalid user_id in token"})
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Set("email", claims["email"])
		c.Set("role", claims["role"])
		c.Next()
	}
}

func GenerateToken(userID uint, email, role string, cfg *config.Config) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"role":    role,
		"exp":     time.Now().Add(cfg.JWT.Expiry).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWT.Secret))
}

func RoleMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			utils.ErrorResponse(c, 403, gin.H{"message": "Forbidden: role not found"})
			c.Abort()
			return
		}

		roleStr, ok := role.(string)
		if !ok {
			utils.ErrorResponse(c, 403, gin.H{"message": "Forbidden: invalid role type"})
			c.Abort()
			return
		}

		for _, allowedRole := range allowedRoles {
			if roleStr == allowedRole {
				c.Next()
				return
			}
		}

		utils.ErrorResponse(c, 403, gin.H{"message": "Forbidden: insufficient permissions"})
		c.Abort()
	}
}
