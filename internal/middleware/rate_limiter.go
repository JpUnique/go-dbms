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

// NewRateLimiter returns an IP-based rate limiter with its own private
// state, so multiple instances (e.g. a tighter limiter on 2FA/reset
// endpoints alongside a looser default elsewhere) don't share counters.
func NewRateLimiter(limit int, window time.Duration) gin.HandlerFunc {

	clients := make(map[string]*clientLimiter)
	var mu sync.Mutex

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

// RateLimitMiddleware is the default general-purpose limiter (20 req/min/IP).
func RateLimitMiddleware() gin.HandlerFunc {
	return NewRateLimiter(20, time.Minute)
}
