package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/npinot/vibe/backend/internal/config"
	"github.com/npinot/vibe/backend/internal/model"
	"github.com/npinot/vibe/backend/internal/repository"
)

type AuthMiddleware struct {
	cfg      *config.Config
	userRepo repository.UserRepository
}

func NewAuthMiddleware(cfg *config.Config, userRepo repository.UserRepository) *AuthMiddleware {
	return &AuthMiddleware{
		cfg:      cfg,
		userRepo: userRepo,
	}
}

func (m *AuthMiddleware) JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authorization header"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		tokenString := parts[1]

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(m.cfg.JWTSecret), nil
		})

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		if !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token is not valid"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}

		userIDStr, ok := claims["user_id"].(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token"})
			c.Abort()
			return
		}

		ctx := context.Background()
		user, err := m.userRepo.FindByID(ctx, userIDStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			c.Abort()
			return
		}

		c.Set("currentUser", user)
		c.Next()
	}
}

func GetCurrentUser(c *gin.Context) (*model.User, error) {
	user, exists := c.Get("currentUser")
	if !exists {
		return nil, fmt.Errorf("user not found in context")
	}

	currentUser, ok := user.(*model.User)
	if !ok {
		return nil, fmt.Errorf("invalid user type in context")
	}

	return currentUser, nil
}

// GetCurrentUserID extracts the user ID from the context
func GetCurrentUserID(c *gin.Context) uuid.UUID {
	user, err := GetCurrentUser(c)
	if err != nil {
		return uuid.Nil
	}
	return user.ID
}
