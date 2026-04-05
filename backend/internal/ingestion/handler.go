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

	// Return realistic responses to keep bots engaged
	w.Header().Set("Server", "nginx/1.24.0")
	switch r.URL.Path {
	case "/robots.txt":
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(robotsTxt))
	case "/sitemap.xml":
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(sitemapXML))
	case "/.well-known/security.txt":
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(securityTxt))
	default:
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(trapHTML))
	}
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

// robots.txt — "disallowed" paths are magnets for malicious bots
const robotsTxt = `User-agent: *
Disallow: /admin/
Disallow: /wp-admin/
Disallow: /wp-login.php
Disallow: /administrator/
Disallow: /backup/
Disallow: /config/
Disallow: /database/
Disallow: /api/v1/users
Disallow: /api/v1/keys
Disallow: /.env
Disallow: /.git/
Disallow: /debug/
Disallow: /server-status
Disallow: /phpmyadmin/
Disallow: /cpanel/
Disallow: /user/login
Disallow: /xmlrpc.php

Sitemap: /sitemap.xml
`

const sitemapXML = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>/</loc><priority>1.0</priority></url>
  <url><loc>/about</loc><priority>0.8</priority></url>
  <url><loc>/contact</loc><priority>0.8</priority></url>
  <url><loc>/blog</loc><priority>0.7</priority></url>
  <url><loc>/products</loc><priority>0.7</priority></url>
  <url><loc>/login</loc><priority>0.6</priority></url>
  <url><loc>/register</loc><priority>0.6</priority></url>
  <url><loc>/api/docs</loc><priority>0.5</priority></url>
  <url><loc>/user/profile</loc><priority>0.5</priority></url>
  <url><loc>/search</loc><priority>0.5</priority></url>
</urlset>`

const securityTxt = `Contact: mailto:admin@example.com
Preferred-Languages: en
`

const trapHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Welcome — My Website</title>
  <meta name="description" content="Official company website with products, blog, and customer portal.">
  <meta name="keywords" content="products, services, login, account, dashboard, api">
</head>
<body>
  <nav>
    <a href="/">Home</a>
    <a href="/about">About</a>
    <a href="/blog">Blog</a>
    <a href="/products">Products</a>
    <a href="/contact">Contact</a>
    <a href="/login">Login</a>
    <a href="/register">Register</a>
  </nav>
  <h1>Welcome to Our Website</h1>
  <p>We offer a wide range of products and services. Please log in to access your account.</p>
  <h2>Quick Links</h2>
  <ul>
    <li><a href="/user/profile">My Account</a></li>
    <li><a href="/api/docs">API Documentation</a></li>
    <li><a href="/search?q=">Search</a></li>
    <li><a href="/blog/2024/getting-started">Getting Started Guide</a></li>
  </ul>
  <!-- TODO: remove before production -->
  <!-- <a href="/admin/">Admin Panel</a> -->
  <!-- <a href="/debug/">Debug Console</a> -->
  <!-- staging: /api/v1/internal -->
  <footer>
    <a href="/terms">Terms</a> | <a href="/privacy">Privacy</a> | <a href="/sitemap.xml">Sitemap</a>
  </footer>
</body>
</html>`
