package utils

import (
	"time"
)

// RateLimiter limits how often certain operations can happen
type RateLimiter struct {
	ticker *time.Ticker
	done   chan bool
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(interval time.Duration) *RateLimiter {
	return &RateLimiter{
		ticker: time.NewTicker(interval),
		done:   make(chan bool),
	}
}

// Wait blocks until the next interval
func (rl *RateLimiter) Wait() {
	select {
	case <-rl.ticker.C:
		// Time to proceed
	case <-rl.done:
		// Rate limiter was stopped
	}
}

// Stop stops the rate limiter
func (rl *RateLimiter) Stop() {
	rl.ticker.Stop()
	close(rl.done)
}

// MaxInt returns the maximum of two integers
func MaxInt(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}