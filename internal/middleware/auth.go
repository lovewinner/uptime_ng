package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"uptime_ng/internal/config"
	"uptime_ng/internal/model"
)

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			abortWithError(c, http.StatusUnauthorized, "missing_authorization_header", "missing authorization header")
			return
		}

		claims, err := parseJWT(strings.TrimPrefix(authHeader, "Bearer "))
		if err != nil {
			abortWithError(c, http.StatusUnauthorized, "invalid_token", "invalid token")
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func WSAuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := c.Query("token")
		if tokenStr == "" {
			authHeader := c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}
		if tokenStr == "" {
			abortWithError(c, http.StatusUnauthorized, "missing_token", "missing token")
			return
		}

		claims, err := parseJWT(tokenStr)
		if err != nil {
			abortWithError(c, http.StatusUnauthorized, "invalid_token", "invalid token")
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func parseJWT(tokenStr string) (*model.JWTClaims, error) {
	claims := &model.JWTClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %s", token.Header["alg"])
		}
		return []byte(config.AppConfig.JWT.Secret), nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists || role != model.RoleAdmin {
			abortWithError(c, http.StatusForbidden, "admin_required", "admin role required")
			return
		}
		c.Next()
	}
}

func abortWithError(c *gin.Context, status int, code string, message string) {
	c.AbortWithStatusJSON(status, gin.H{
		"error": message,
		"code":  code,
	})
}
