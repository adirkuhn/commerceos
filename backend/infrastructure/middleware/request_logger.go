package middleware

import (
	"bytes"
	"io"
	"log"

	"github.com/gin-gonic/gin"
)

// requestLogBodyMaxBytes caps how much of a body ends up in the log line
// itself -- purely a log-line size guard, not a request-size limit; the
// handler downstream still receives every byte of the original body.
const requestLogBodyMaxBytes = 8 * 1024

// RequestBodyLogger logs every request's method, full URL (path + query
// string), and body via the standard logger. Reads the body, then
// restores it behind a fresh io.NopCloser so downstream handlers and the
// OpenAPI request validator both still see the original bytes -- an
// http.Request's Body can only be read once otherwise.
func RequestBodyLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body []byte
		if c.Request.Body != nil {
			body, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewReader(body))
		}

		logged, suffix := body, ""
		if len(logged) > requestLogBodyMaxBytes {
			logged = logged[:requestLogBodyMaxBytes]
			suffix = "...(truncated)"
		}

		log.Printf("request: %s %s body=%s%s", c.Request.Method, c.Request.URL.String(), logged, suffix)

		c.Next()
	}
}
