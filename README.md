# BOTLOG - Real-Time Bot Traffic Monitor

A standalone honeypot that logs and visualizes bot/scanner traffic in real-time.

## Architecture

- **Go backend** — dual HTTP server (trap on `:8080`, API on `:8081`)
- **ClickHouse** — columnar analytics DB for billions of rows
- **React + TypeScript** — responsive dashboard with dark/tech theme
- **MaxMind GeoLite2** — IP to city/country resolution

## Quick Start

### 1. Get GeoLite2 Database

Register at [MaxMind](https://dev.maxmind.com/geoip/geolite2-free-geolocation-data) and download `GeoLite2-City.mmdb`.

```bash
mkdir -p data
# Place GeoLite2-City.mmdb in ./data/
```

### 2. Deploy with Docker Compose

```bash
docker compose up -d
```

This starts:
- **ClickHouse** on ports 8123 (HTTP) / 9000 (native)
- **Backend** trap on port 8080 (point your DNS/firewall here)
- **Frontend** dashboard on port 3000

### 3. Access Dashboard

Open `http://your-server:3000` to see:
- **LIVE** — real-time scrolling feed of bot hits
- **STATS** — filterable charts (bar/line/area/pie) + CSV export

## Configuration

Environment variables for the backend container:

| Variable | Default | Description |
|----------|---------|-------------|
| `TRAP_PORT` | `8080` | Port for the honeypot trap |
| `API_PORT` | `8081` | Port for the dashboard API |
| `CLICKHOUSE_ADDR` | `127.0.0.1:9000` | ClickHouse native address |
| `CLICKHOUSE_DB` | `botlog` | Database name |
| `GEOIP_PATH` | `/data/GeoLite2-City.mmdb` | Path to MaxMind DB |

## Development

```bash
# Backend
cd backend && go run ./cmd/main.go

# Frontend
cd frontend && npm install && npm run dev
```
