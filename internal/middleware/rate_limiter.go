package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiterConfig configures rate limiting behavior
type RateLimiterConfig struct {
	RequestsPerSecond float64
	Burst             int
}

// rateLimiterMap stores rate limiters per IP address
type rateLimiterMap struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	config   RateLimiterConfig
}

// NewRateLimiterMap creates a new rate limiter map
func NewRateLimiterMap(config RateLimiterConfig) *rateLimiterMap {
	rl := &rateLimiterMap{
		limiters: make(map[string]*rate.Limiter),
		config:   config,
	}

	// Clean up old limiters periodically
	go rl.cleanup()

	return rl
}

// getLimiter returns or creates a rate limiter for the given IP
func (rl *rateLimiterMap) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[ip]
	if !exists {
		limiter = rate.NewLimiter(rate.Limit(rl.config.RequestsPerSecond), rl.config.Burst)
		rl.limiters[ip] = limiter
	}

	return limiter
}

// cleanup removes old limiters periodically to prevent memory leaks
func (rl *rateLimiterMap) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		// Keep only active limiters (those that have tokens available)
		// In a production environment, you might want more sophisticated cleanup
		if len(rl.limiters) > 1000 {
			// Reset if too many limiters accumulated
			rl.limiters = make(map[string]*rate.Limiter)
		}
		rl.mu.Unlock()
	}
}

// RateLimiterMiddleware creates a rate limiting middleware
func RateLimiterMiddleware(config RateLimiterConfig) gin.HandlerFunc {
	limiterMap := NewRateLimiterMap(config)

	return func(c *gin.Context) {
		// Get client IP
		ip := c.ClientIP()

		// Get or create limiter for this IP
		limiter := limiterMap.getLimiter(ip)

		// Check if request is allowed
		if !limiter.Allow() {
			reservation := limiter.Reserve()
			retryAfter := reservation.DelayFrom(time.Now()).Seconds()
			reservation.Cancel() // Cancel the reservation since we're rejecting the request
			
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":      "Rate limit exceeded. Please try again later.",
				"retry_after": retryAfter,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
