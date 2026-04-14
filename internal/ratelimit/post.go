package ratelimit

import (
	"net"
	"net/http"
	"sync"
	"time"
)

const PostMinInterval = 5 * time.Second

var mu sync.Mutex
var lastByIP = map[string]time.Time{}

// ClientIP returns host without port (только RemoteAddr; без X-Forwarded-For, чтобы не обходили лимит).
func ClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// AllowPost reports whether a new post/thread is allowed from this IP.
func AllowPost(r *http.Request) (ok bool, retryAfter time.Duration) {
	ip := ClientIP(r)
	now := time.Now()
	mu.Lock()
	defer mu.Unlock()
	if t, ok := lastByIP[ip]; ok {
		if d := now.Sub(t); d < PostMinInterval {
			return false, PostMinInterval - d
		}
	}
	return true, 0
}

// RecordPost marks successful submission time for the IP.
func RecordPost(r *http.Request) {
	ip := ClientIP(r)
	mu.Lock()
	lastByIP[ip] = time.Now()
	mu.Unlock()
}
