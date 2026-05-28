package checker

import (
	"net"
	"time"
)

// connectivityAnchors are reliable, always-on TCP endpoints used to decide
// whether the local system has working internet access. We dial several so a
// single anchor outage doesn't produce a false "offline" result.
var connectivityAnchors = []string{
	"1.1.1.1:443", // Cloudflare
	"8.8.8.8:53",  // Google DNS
	"9.9.9.9:443", // Quad9
}

// probeOnline reports whether at least one anchor is reachable within timeout.
// It is overridable in tests.
var probeOnline = func(timeout time.Duration) bool {
	for _, addr := range connectivityAnchors {
		conn, err := net.DialTimeout("tcp", addr, timeout)
		if err == nil {
			_ = conn.Close()
			return true
		}
	}
	return false
}

// IsOnline reports whether the system currently has internet connectivity.
func IsOnline(timeout time.Duration) bool {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	return probeOnline(timeout)
}
