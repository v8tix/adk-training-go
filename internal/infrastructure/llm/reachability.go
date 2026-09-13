package llm

import (
	"net"
	"net/url"
	"time"
)

// OllamaReachable dials cfg.OllamaBaseURL's host:port directly, rather than
// a second hardcoded address, so this check can never drift from the URL
// BuildModel actually uses. Every cmd/ program's integration test shares
// this instead of repeating it: a TCP dial is enough to distinguish "server
// not running" (skip the test) from a real model-response failure (fail
// it).
func OllamaReachable(cfg Config, timeout time.Duration) bool {
	u, err := url.Parse(cfg.OllamaBaseURL)
	if err != nil {
		return false
	}
	conn, err := net.DialTimeout("tcp", u.Host, timeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
