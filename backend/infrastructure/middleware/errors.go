package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type APIResponse struct {
	Data  any       `json:"data"`
	Error *APIError `json:"error"`
}

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, APIResponse{Data: data})
}

func Fail(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, APIResponse{Error: &APIError{Code: code, Message: message}})
}

// ErrorHandler maps domain errors to HTTP responses. Register as gin middleware.
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) == 0 {
			return
		}
		err := c.Errors.Last().Err
		switch {
		case errors.Is(err, ErrNotFound):
			Fail(c, http.StatusNotFound, "NOT_FOUND", err.Error())
		case errors.Is(err, ErrUnprocessable):
			Fail(c, http.StatusUnprocessableEntity, "UNPROCESSABLE", err.Error())
		default:
			Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
		}
	}
}

var (
	ErrNotFound      = errors.New("resource not found")
	ErrUnprocessable = errors.New("unprocessable entity")
)
