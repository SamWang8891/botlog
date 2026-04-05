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
	ch             *ch.Client
	geo            *geoip.Resolver
	trustedProxies []*net.IPNet
}

func NewHandler(chClient *ch.Client, geoResolver *geoip.Resolver, trustedProxies []*net.IPNet) *Handler {
	return &Handler{ch: chClient, geo: geoResolver, trustedProxies: trustedProxies}
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
		if r.ContentLength > bodySize {
			bodySize = r.ContentLength
		}
		r.Body.Close()
	}

	// Extract client IP (trust X-Forwarded-For only from known proxies)
	ip := h.extractIP(r)

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

func (h *Handler) extractIP(r *http.Request) string {
	remoteIP := remoteAddrIP(r)

	// Only trust forwarded headers if the direct connection is from a known proxy
	if h.isTrustedProxy(remoteIP) {
		// Walk X-Forwarded-For right-to-left to find the first non-proxy IP
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			for i := len(parts) - 1; i >= 0; i-- {
				candidate := strings.TrimSpace(parts[i])
				if candidate == "" {
					continue
				}
				if !h.isTrustedProxy(net.ParseIP(candidate)) {
					return candidate
				}
			}
		}
		if xri := r.Header.Get("X-Real-Ip"); xri != "" {
			return xri
		}
	}

	if remoteIP != nil {
		return remoteIP.String()
	}
	return r.RemoteAddr
}

func (h *Handler) isTrustedProxy(ip net.IP) bool {
	if ip == nil {
		return false
	}
	for _, cidr := range h.trustedProxies {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

func remoteAddrIP(r *http.Request) net.IP {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return net.ParseIP(r.RemoteAddr)
	}
	return net.ParseIP(host)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
