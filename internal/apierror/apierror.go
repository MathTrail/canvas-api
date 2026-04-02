// Package apierror defines shared HTTP error response types.
package apierror

import "github.com/gin-gonic/gin"

// Response represents a structured HTTP error response.
type Response struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Abort writes a structured JSON error response and aborts the request chain.
func Abort(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, Response{Code: code, Message: message})
}
