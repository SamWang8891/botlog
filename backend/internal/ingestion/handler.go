package ingestion

import (
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	ch "github.com/samwang8891/whats-the-bot-doing/internal/clickhouse"
	"github.com/samwang8891/whats-the-bot-doing/internal/geoip"
)

const maxBodyPreview = 4096

type Handler struct {
	ch  *ch.Client
	geo *geoip.Resolver
}

func NewHandler(chClient *ch.Client, geoResolver *geoip.Resolver) *Handler {
	return &Handler{ch: chClient, geo: geoResolver}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()

	// Read body preview
	var bodyPreview string
	var bodySize int64
	if r.Body != nil {
		limited := io.LimitReader(r.Body, maxBodyPreview+1)
		data, err := io.ReadAll(limited)
		if err == nil {
			bodySize = int64(len(data))
			if len(data) > maxBodyPreview {
				bodyPreview = string(data[:maxBodyPreview]) + "..."
			} else {
				bodyPreview = string(data)
			}
		}
		// Try to get actual content-length if larger
		if r.ContentLength > bodySize {
			bodySize = r.ContentLength
		}
		r.Body.Close()
	}

	// Extract client IP
	ip := extractIP(r)

	// GeoIP lookup
	loc := h.geo.Lookup(ip)

	// Collect interesting headers
	headers := make(map[string]string)
	for _, name := range []string{
		"Accept", "Accept-Language", "Accept-Encoding",
		"Referer", "Origin", "X-Forwarded-For",
		"X-Real-Ip", "Cookie",
	} {
		if v := r.Header.Get(name); v != "" {
			headers[name] = v
		}
	}

	hit := ch.Hit{
		Timestamp:   now,
		Method:      r.Method,
		Path:        r.URL.Path,
		UserAgent:   r.UserAgent(),
		Country:     loc.Country,
		City:        loc.City,
		ContentType: r.Header.Get("Content-Type"),
		BodyPreview: bodyPreview,
		BodySize:    bodySize,
		Headers:     headers,
	}

	h.ch.Insert(hit)

	log.Printf("HIT %s %s from %s (%s, %s) UA=%s",
		r.Method, r.URL.Path, ip, loc.City, loc.Country, truncate(r.UserAgent(), 60))

	// Return a realistic-looking response to keep bots engaged
	w.Header().Set("Server", "nginx/1.24.0")
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<!DOCTYPE html><html><head><title>Welcome</title></head><body><h1>Welcome</h1></body></html>`))
}

func extractIP(r *http.Request) string {
	// Check X-Forwarded-For first (for reverse proxy setups)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.SplitN(xff, ",", 2)
		ip := strings.TrimSpace(parts[0])
		if ip != "" {
			return ip
		}
	}
	if xri := r.Header.Get("X-Real-Ip"); xri != "" {
		return xri
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
