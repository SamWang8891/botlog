package config

import (
	"os"
)

type Config struct {
	TrapPort       string
	APIPort        string
	ClickHouseAddr string
	ClickHouseDB   string
	GeoIPPath      string
	BatchSize      int
	FlushInterval  int // milliseconds
}

func Load() *Config {
	return &Config{
		TrapPort:       getEnv("TRAP_PORT", "8080"),
		APIPort:        getEnv("API_PORT", "8081"),
		ClickHouseAddr: getEnv("CLICKHOUSE_ADDR", "127.0.0.1:9000"),
		ClickHouseDB:   getEnv("CLICKHOUSE_DB", "botlog"),
		GeoIPPath:      getEnv("GEOIP_PATH", "/data/GeoLite2-City.mmdb"),
		BatchSize:      1000,
		FlushInterval:  1000,
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
