# BOTLOG - Real-Time Bot Traffic Monitor

A standalone honeypot that logs and visualizes bot/scanner traffic in real-time.

## Architecture

- **Go backend** — trap + API server behind nginx
- **ClickHouse** — columnar analytics DB for billions of rows
- **React + TypeScript** — responsive dashboard with dark/tech theme
- **MaxMind GeoLite2** — IP to city/country resolution

## Quick Start

```bash
git clone https://github.com/SamWang8891/whats-the-bot-doing.git
cd whats-the-bot-doing
./setup.sh
```

That's it. The setup script will:
1. Verify Docker and Docker Compose are installed and running
2. Download the latest GeoLite2-City.mmdb automatically
3. Build and deploy all services
4. Health-check and print status when ready

Everything runs on a **single port (80)**. Point your domain here.

### Routes

| Path | What it does |
|------|-------------|
| `/` | Dashboard — live feed of bot hits |
| `/stats` | Statistics — filterable charts + CSV export |
| `/api/*` | REST API + SSE stream (used by the dashboard) |
| `/*` (anything else) | Bot trap — logs the hit and returns a fake page |

## Updating

```bash
./update.sh
```

This will:
1. Pull the latest code (`git pull --ff-only`)
2. Re-download the latest GeoLite2-City.mmdb
3. Rebuild and restart all services with zero data loss

## Reverse Proxy Setup (Nginx Proxy Manager)

If you run BOTLOG behind a reverse proxy (e.g., Nginx Proxy Manager), you need to:

### 1. Add your proxy IP to `proxies.conf`

Edit `proxies.conf` in the project root and add your proxy's IP:

```
# Nginx Proxy Manager IP
192.168.1.100
```

This tells BOTLOG to trust `X-Forwarded-For` headers from that IP so it logs the **real** client IP, not your proxy's IP. Supports IPs and CIDRs (`10.0.0.0/8`).

### 2. Configure your reverse proxy

In Nginx Proxy Manager, create a proxy host pointing to `http://<botlog-server>:8120` (or whatever port you set) and enable:

- **WebSocket support** — needed for SSE live feed
- Under **Advanced**, add:

```nginx
proxy_buffering off;
proxy_cache off;
proxy_read_timeout 86400s;
proxy_set_header X-Real-IP $remote_addr;
proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
```

These prevent the proxy from buffering the SSE stream and ensure real client IPs are forwarded.

### 3. Restart

```bash
docker compose up -d --build backend
```

## Configuration

Environment variables for the backend container:

| Variable | Default | Description |
|----------|---------|-------------|
| `TRAP_PORT` | `8080` | Port for the honeypot trap |
| `API_PORT` | `8081` | Port for the dashboard API |
| `CLICKHOUSE_ADDR` | `127.0.0.1:9000` | ClickHouse native address |
| `CLICKHOUSE_DB` | `botlog` | Database name |
| `GEOIP_PATH` | `/data/GeoLite2-City.mmdb` | Path to MaxMind DB |
| `PROXIES_PATH` | `/data/proxies.conf` | Trusted proxy list |

## Development

```bash
# Backend
cd backend && go run ./cmd/main.go

# Frontend
cd frontend && npm install && npm run dev
```
