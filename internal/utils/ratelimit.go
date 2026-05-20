package utils

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
	limit   = 10          // requests
	window  = time.Minute // per minute
)

// RateLimitMiddleware basic limiter
func RateLimitMiddleware() gin.HandlerFunc {

	return func(c *gin.Context) {

		ip, _, _ := net.SplitHostPort(c.Request.RemoteAddr)

		mu.Lock()
		defer mu.Unlock()

		cl, exists := clients[ip]
		if !exists {
			clients[ip] = &clientLimiter{count: 1, lastReset: time.Now()}
		} else {

			if time.Since(cl.lastReset) > window {
				cl.count = 0
				cl.lastReset = time.Now()
			}

			cl.count++

			if cl.count > limit {
				c.AbortWithStatusJSON(429, gin.H{
					"error": "too many requests",
				})
				return
			}
		}

		c.Next()
	}
}
