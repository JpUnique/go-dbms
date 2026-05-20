package middleware

import (
	"net"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type clientLimiter struct {
	count     int
	lastReset time.Time
}

var (
	clients = make(map[string]*clientLimiter)
	mu      sync.Mutex
	limit   = 20 // requests per window
	window  = time.Minute
)

// RateLimitMiddleware basic protection
func RateLimitMiddleware() gin.HandlerFunc {

	return func(c *gin.Context) {

		ip, _, _ := net.SplitHostPort(c.Request.RemoteAddr)

		mu.Lock()
		defer mu.Unlock()

		client, exists := clients[ip]

		if !exists {
			clients[ip] = &clientLimiter{
				count:     1,
				lastReset: time.Now(),
			}
		} else {

			if time.Since(client.lastReset) > window {
				client.count = 0
				client.lastReset = time.Now()
			}

			client.count++

			if client.count > limit {
				c.AbortWithStatusJSON(429, gin.H{
					"error": "too many requests",
				})
				return
			}
		}

		c.Next()
	}
}
