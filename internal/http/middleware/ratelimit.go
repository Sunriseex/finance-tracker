package middleware

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type rateLimitBucket struct {
	windowStart time.Time
	count       int
}

func RateLimitByIP(limit int, window time.Duration) func(http.Handler) http.Handler {
	var (
		mu      sync.Mutex
		buckets = make(map[string]rateLimitBucket)
		now     = time.Now
	)

	return rateLimitByIP(limit, window, now, &mu, buckets)
}

func rateLimitByIP(limit int, window time.Duration, now func() time.Time, mu *sync.Mutex, buckets map[string]rateLimitBucket) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if limit <= 0 || window <= 0 {
				next.ServeHTTP(w, r)
				return
			}

			key := clientIP(r)
			current := now()

			mu.Lock()
			bucket := buckets[key]
			if bucket.windowStart.IsZero() || current.Sub(bucket.windowStart) >= window {
				bucket = rateLimitBucket{windowStart: current}
			}
			bucket.count++
			buckets[key] = bucket
			allowed := bucket.count <= limit
			if len(buckets) > limit*8 {
				for bucketKey, bucket := range buckets {
					if current.Sub(bucket.windowStart) >= window {
						delete(buckets, bucketKey)
					}
				}
			}
			mu.Unlock()

			if !allowed {
				w.Header().Set("Retry-After", strconv.Itoa(max(1, int(window.Seconds()))))
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func clientIP(r *http.Request) string {
	if forwardedFor := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwardedFor != "" {
		ip, _, _ := strings.Cut(forwardedFor, ",")
		if ip = strings.TrimSpace(ip); ip != "" {
			return ip
		}
	}
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}
