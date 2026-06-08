package middleware

import (
	"github.com/gin-gonic/gin"
)

const adminContentSecurityPolicy = "default-src 'self'; base-uri 'self'; object-src 'none'; frame-ancestors 'none'; img-src 'self' data: blob:; font-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self'; connect-src 'self' ws: wss:"

// AdminSecurityHeaders sets the admin panel's security headers. isSecure reports
// whether the request arrived over HTTPS and gates the HSTS header; callers
// inject a trusted-proxy-aware check (e.g. api.RequestIsHTTPS) so a spoofed
// X-Forwarded-Proto from an untrusted client cannot trigger HSTS. When isSecure
// is nil, only a real TLS connection is treated as secure.
func AdminSecurityHeaders(isSecure func(*gin.Context) bool) gin.HandlerFunc {
	if isSecure == nil {
		isSecure = func(c *gin.Context) bool { return c.Request.TLS != nil }
	}
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Content-Security-Policy", adminContentSecurityPolicy)
		if isSecure(c) {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		c.Next()
	}
}

func SubSecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Cache-Control", "no-store")
		c.Next()
	}
}
