package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type requestValidationError struct {
	code    string
	message string
}

func errorResponse(c *gin.Context, status int, code string, message string) {
	c.JSON(status, gin.H{
		"error": message,
		"code":  code,
	})
}

func badRequest(c *gin.Context, code string, message string) {
	errorResponse(c, http.StatusBadRequest, code, message)
}

func lookupErrorResponse(c *gin.Context, err error, notFoundCode string, notFoundMessage string, failureCode string) {
	if isRecordNotFound(err) {
		errorResponse(c, http.StatusNotFound, notFoundCode, notFoundMessage)
		return
	}
	errorResponse(c, http.StatusInternalServerError, failureCode, err.Error())
}

func isRecordNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
