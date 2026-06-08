package middleware

import (
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/idna"
)

func DomainValidator(domain string) gin.HandlerFunc {
	// Pre-normalize the configured domain to its ASCII (punycode) form so that an
	// IDN domain (e.g. "панель.ru") matches the punycode Host header browsers
	// send. Falls back to the raw value if conversion fails (fail-closed safe).
	expected := domain
	if ascii, err := idna.ToASCII(domain); err == nil && ascii != "" {
		expected = ascii
	}
	return func(c *gin.Context) {
		host := c.Request.Host
		if splitHost, _, err := net.SplitHostPort(c.Request.Host); err == nil {
			host = splitHost
		} else {
			host = strings.Trim(host, "[]")
		}
		if ascii, err := idna.ToASCII(host); err == nil && ascii != "" {
			host = ascii
		}

		if !strings.EqualFold(host, expected) {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		c.Next()
	}
}
