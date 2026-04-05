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

func loadProxies(path string) []*net.IPNet {
	f, err := os.Open(path)
	if err != nil {
		log.Printf("No proxy config at %s (using RemoteAddr only): %v", path, err)
		return nil
	}
	defer f.Close()

	var nets []*net.IPNet
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
	}
	if len(nets) > 0 {
		log.Printf("Loaded %d trusted proxy entries", len(nets))
	}
	return nets
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
