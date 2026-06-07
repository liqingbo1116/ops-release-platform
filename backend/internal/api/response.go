package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
	RequestID string `json:"requestId"`
}

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{
		Code:      "OK",
		Message:   "success",
		Data:      data,
		RequestID: requestID(),
	})
}

func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Response{
		Code:      "OK",
		Message:   "success",
		Data:      data,
		RequestID: requestID(),
	})
}

func NotFound(c *gin.Context, message string) {
	c.JSON(http.StatusNotFound, Response{
		Code:      "NOT_FOUND",
		Message:   message,
		RequestID: requestID(),
	})
}

func BadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, Response{
		Code:      "VALIDATION_ERROR",
		Message:   message,
		RequestID: requestID(),
	})
}

func requestID() string {
	return fmt.Sprintf("req-%s", time.Now().Format("20060102-150405.000000"))
}
