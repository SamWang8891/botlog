package config

import (
	"bufio"
	"log"
	"net"
	"os"
	"strings"
)

type Config struct {
	TrapPort       string
	APIPort        string
	ClickHouseAddr string
	ClickHouseDB   string
	GeoIPPath      string
	ProxiesPath    string
	TrustedProxies []*net.IPNet
	BatchSize      int
	FlushInterval  int // milliseconds
}

func Load() *Config {
	cfg := &Config{
		TrapPort:       getEnv("TRAP_PORT", "8080"),
		APIPort:        getEnv("API_PORT", "8081"),
		ClickHouseAddr: getEnv("CLICKHOUSE_ADDR", "127.0.0.1:9000"),
		ClickHouseDB:   getEnv("CLICKHOUSE_DB", "botlog"),
		GeoIPPath:      getEnv("GEOIP_PATH", "/data/GeoLite2-City.mmdb"),
		ProxiesPath:    getEnv("PROXIES_PATH", "/data/proxies.conf"),
		BatchSize:      1000,
		FlushInterval:  1000,
	}
	cfg.TrustedProxies = loadProxies(cfg.ProxiesPath)
	return cfg
}

// privateRanges are always trusted — they cover Docker internal networks
// and any local reverse proxy (Nginx Proxy Manager, Caddy, etc.).
var privateRanges = []string{
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"127.0.0.0/8",
	"::1/128",
	"fc00::/7",
}

func loadProxies(path string) []*net.IPNet {
	var nets []*net.IPNet

	// Always trust private/loopback ranges (Docker, local proxies)
	for _, cidr := range privateRanges {
		_, n, _ := net.ParseCIDR(cidr)
		nets = append(nets, n)
	}

	// Load additional user-defined proxies from file
	f, err := os.Open(path)
	if err != nil {
		log.Printf("No proxy config at %s (using private ranges only): %v", path, err)
		return nets
	}
	defer f.Close()

	extra := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Add /32 or /128 if no CIDR notation
		if !strings.Contains(line, "/") {
			if strings.Contains(line, ":") {
				line += "/128"
			} else {
				line += "/32"
			}
		}
		_, cidr, err := net.ParseCIDR(line)
		if err != nil {
			log.Printf("Invalid proxy entry %q: %v", line, err)
			continue
		}
		nets = append(nets, cidr)
		extra++
	}
	if extra > 0 {
		log.Printf("Loaded %d extra trusted proxy entries from %s", extra, path)
	}
	return nets
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
