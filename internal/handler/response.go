package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func errorResponse(c *gin.Context, status int, code string, message string) {
	c.JSON(status, gin.H{
		"error": message,
		"code":  code,
	})
}

func badRequest(c *gin.Context, code string, message string) {
	errorResponse(c, http.StatusBadRequest, code, message)
}
