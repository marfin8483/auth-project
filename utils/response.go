package utils

import (
	"auth-api/dto"
	"encoding/json"

	"github.com/gin-gonic/gin"
)

func SuccessResponse(c *gin.Context, code int, data interface{}) {
	response := dto.APIResponse{
		Status: "success",
		Code:   code,
		Data:   data,
	}
	c.JSON(code, response)
}

func ErrorResponse(c *gin.Context, code int, data interface{}) {
	response := dto.APIResponse{
		Status: "error",
		Code:   code,
		Data:   data,
	}
	c.JSON(code, response)
}

// Helper function untuk convert ke JSON string
func ToJSON(v interface{}) string {
	bytes, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(bytes)
}
