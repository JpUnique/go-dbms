package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Success response
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"data":  data,
		"error": nil,
	})
}

// Created response
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, gin.H{
		"data":  data,
		"error": nil,
	})
}

// Error response
func Error(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{
		"data":  nil,
		"error": message,
	})
}
