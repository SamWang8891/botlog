package api

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type Server struct {
	conn driver.Conn
	mux  *http.ServeMux
}

func NewServer(conn driver.Conn) *Server {
	s := &Server{conn: conn}
	s.mux = http.NewServeMux()
	s.mux.HandleFunc("/api/hits/live", s.handleSSE)
	s.mux.HandleFunc("/api/stats/timeline", s.handleTimeline)
	s.mux.HandleFunc("/api/stats/countries", s.handleCountries)
	s.mux.HandleFunc("/api/stats/methods", s.handleMethods)
	s.mux.HandleFunc("/api/stats/endpoints", s.handleEndpoints)
	s.mux.HandleFunc("/api/stats/agents", s.handleAgents)
	s.mux.HandleFunc("/api/stats/overview", s.handleOverview)
	s.mux.HandleFunc("/api/export/csv", s.handleCSVExport)
	s.mux.HandleFunc("/api/filters", s.handleFilters)
	return s
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

// SSE endpoint: streams recent hits every second
func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Tell browser to retry after 3s if connection drops
	fmt.Fprintf(w, "retry: 3000\n\n")
	flusher.Flush()

	// Send initial backfill (latest 100), then poll for new records only
	var cursor time.Time
	pingTick := 0

	// First: send backfill immediately
	{
		rows, err := s.conn.Query(r.Context(), `SELECT timestamp, method, path, user_agent, country, city,
			content_type, body_preview, body_size
			FROM hits ORDER BY timestamp DESC LIMIT 100`)
		if err != nil {
			log.Printf("SSE backfill error: %v", err)
		} else {
			var hits []map[string]interface{}
			for rows.Next() {
				var ts time.Time
				var method, path, ua, country, city, ct, body string
				var bodySize int64
				if err := rows.Scan(&ts, &method, &path, &ua, &country, &city, &ct, &body, &bodySize); err != nil {
					continue
				}
				if ts.After(cursor) {
					cursor = ts
				}
				hits = append(hits, map[string]interface{}{
					"timestamp":    ts.Format(time.RFC3339Nano),
					"method":       method,
					"path":         path,
					"user_agent":   ua,
					"country":      country,
					"city":         city,
					"content_type": ct,
					"body_preview": body,
					"body_size":    bodySize,
				})
			}
			rows.Close()
			if len(hits) > 0 {
				data, _ := json.Marshal(hits)
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			}
		}
	}

	// If no records exist yet, start cursor from now
	if cursor.IsZero() {
		cursor = time.Now().UTC()
	}

	// Poll loop: only fetch records strictly newer than cursor
	for {
		time.Sleep(time.Second)

		select {
		case <-r.Context().Done():
			return
		default:
		}

		rows, err := s.conn.Query(r.Context(), `SELECT timestamp, method, path, user_agent, country, city,
			content_type, body_preview, body_size
			FROM hits WHERE timestamp > @cursor ORDER BY timestamp ASC LIMIT 50`,
			clickhouse.DateNamed("cursor", cursor, 3))
		if err != nil {
			log.Printf("SSE poll error: %v", err)
			continue
		}

		var hits []map[string]interface{}
		for rows.Next() {
			var ts time.Time
			var method, path, ua, country, city, ct, body string
			var bodySize int64
			if err := rows.Scan(&ts, &method, &path, &ua, &country, &city, &ct, &body, &bodySize); err != nil {
				continue
			}
			if ts.After(cursor) {
				cursor = ts
			}
			hits = append(hits, map[string]interface{}{
				"timestamp":    ts.Format(time.RFC3339Nano),
				"method":       method,
				"path":         path,
				"user_agent":   ua,
				"country":      country,
				"city":         city,
				"content_type": ct,
				"body_preview": body,
				"body_size":    bodySize,
			})
		}
		rows.Close()

		if len(hits) > 0 {
			data, _ := json.Marshal(hits)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
			pingTick = 0
		} else {
			pingTick++
			if pingTick >= 15 {
				fmt.Fprintf(w, ": ping\n\n")
				flusher.Flush()
				pingTick = 0
			}
		}
	}
}

type filterParams struct {
	from    time.Time
	to      time.Time
	country string
	method  string
	path    string
	agent   string
}

func parseFilters(r *http.Request) filterParams {
	f := filterParams{}
	if v := r.URL.Query().Get("from"); v != "" {
		f.from, _ = time.Parse(time.RFC3339, v)
	}
	if v := r.URL.Query().Get("to"); v != "" {
		f.to, _ = time.Parse(time.RFC3339, v)
	} else {
		f.to = time.Now().UTC()
	}
	if f.from.IsZero() {
		f.from = f.to.Add(-24 * time.Hour)
	}
	f.country = r.URL.Query().Get("country")
	f.method = r.URL.Query().Get("method")
	f.path = r.URL.Query().Get("path")
	f.agent = r.URL.Query().Get("agent")
	return f
}

func buildWhereClause(f filterParams) (string, []interface{}) {
	clauses := []string{"timestamp >= ?", "timestamp <= ?"}
	args := []interface{}{f.from, f.to}

	if f.country != "" {
		clauses = append(clauses, "country = ?")
		args = append(args, f.country)
	}
	if f.method != "" {
		clauses = append(clauses, "method = ?")
		args = append(args, f.method)
	}
	if f.path != "" {
		clauses = append(clauses, "path LIKE ?")
		args = append(args, "%"+f.path+"%")
	}
	if f.agent != "" {
		clauses = append(clauses, "user_agent LIKE ?")
		args = append(args, "%"+f.agent+"%")
	}

	return "WHERE " + strings.Join(clauses, " AND "), args
}

// Timeline: hits over time
func (s *Server) handleTimeline(w http.ResponseWriter, r *http.Request) {
	f := parseFilters(r)
	where, args := buildWhereClause(f)

	// Determine granularity based on time range
	duration := f.to.Sub(f.from)
	var granularity string
	switch {
	case duration <= 6*time.Hour:
		granularity = "toStartOfMinute(timestamp)"
	case duration <= 72*time.Hour:
		granularity = "toStartOfHour(timestamp)"
	default:
		granularity = "toStartOfDay(timestamp)"
	}

	query := fmt.Sprintf(`SELECT %s AS t, count() AS hits FROM hits %s GROUP BY t ORDER BY t`, granularity, where)

	rows, err := s.conn.Query(r.Context(), query, args...)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	type point struct {
		Time string `json:"time"`
		Hits uint64 `json:"hits"`
	}
	var result []point
	for rows.Next() {
		var t time.Time
		var hits uint64
		if err := rows.Scan(&t, &hits); err != nil {
			continue
		}
		result = append(result, point{Time: t.Format(time.RFC3339), Hits: hits})
	}
	jsonResponse(w, result)
}

// Countries: pie chart data
func (s *Server) handleCountries(w http.ResponseWriter, r *http.Request) {
	f := parseFilters(r)
	where, args := buildWhereClause(f)
	limit := intParam(r, "limit", 20)

	query := fmt.Sprintf(`SELECT country, count() AS hits FROM hits %s GROUP BY country ORDER BY hits DESC LIMIT %d`, where, limit)
	rows, err := s.conn.Query(r.Context(), query, args...)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	type entry struct {
		Name  string `json:"name"`
		Value uint64 `json:"value"`
	}
	var result []entry
	for rows.Next() {
		var name string
		var val uint64
		if err := rows.Scan(&name, &val); err != nil {
			continue
		}
		result = append(result, entry{Name: name, Value: val})
	}
	jsonResponse(w, result)
}

// Methods: pie chart data
func (s *Server) handleMethods(w http.ResponseWriter, r *http.Request) {
	f := parseFilters(r)
	where, args := buildWhereClause(f)

	query := fmt.Sprintf(`SELECT method, count() AS hits FROM hits %s GROUP BY method ORDER BY hits DESC`, where)
	rows, err := s.conn.Query(r.Context(), query, args...)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	type entry struct {
		Name  string `json:"name"`
		Value uint64 `json:"value"`
	}
	var result []entry
	for rows.Next() {
		var name string
		var val uint64
		if err := rows.Scan(&name, &val); err != nil {
			continue
		}
		result = append(result, entry{Name: name, Value: val})
	}
	jsonResponse(w, result)
}

// Top endpoints
func (s *Server) handleEndpoints(w http.ResponseWriter, r *http.Request) {
	f := parseFilters(r)
	where, args := buildWhereClause(f)
	limit := intParam(r, "limit", 20)

	query := fmt.Sprintf(`SELECT path, count() AS hits FROM hits %s GROUP BY path ORDER BY hits DESC LIMIT %d`, where, limit)
	rows, err := s.conn.Query(r.Context(), query, args...)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	type entry struct {
		Name  string `json:"name"`
		Value uint64 `json:"value"`
	}
	var result []entry
	for rows.Next() {
		var name string
		var val uint64
		if err := rows.Scan(&name, &val); err != nil {
			continue
		}
		result = append(result, entry{Name: name, Value: val})
	}
	jsonResponse(w, result)
}

// Top user agents
func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
	f := parseFilters(r)
	where, args := buildWhereClause(f)
	limit := intParam(r, "limit", 20)

	query := fmt.Sprintf(`SELECT user_agent, count() AS hits FROM hits %s GROUP BY user_agent ORDER BY hits DESC LIMIT %d`, where, limit)
	rows, err := s.conn.Query(r.Context(), query, args...)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	type entry struct {
		Name  string `json:"name"`
		Value uint64 `json:"value"`
	}
	var result []entry
	for rows.Next() {
		var name string
		var val uint64
		if err := rows.Scan(&name, &val); err != nil {
			continue
		}
		result = append(result, entry{Name: name, Value: val})
	}
	jsonResponse(w, result)
}

// Overview stats
func (s *Server) handleOverview(w http.ResponseWriter, r *http.Request) {
	f := parseFilters(r)
	where, args := buildWhereClause(f)

	type overview struct {
		TotalHits       uint64 `json:"total_hits"`
		UniqueCountries uint64 `json:"unique_countries"`
		UniquePaths     uint64 `json:"unique_paths"`
		UniqueAgents    uint64 `json:"unique_agents"`
		WithBody        uint64 `json:"with_body"`
	}

	query := fmt.Sprintf(`SELECT
		count() AS total_hits,
		uniq(country) AS unique_countries,
		uniq(path) AS unique_paths,
		uniq(user_agent) AS unique_agents,
		countIf(body_size > 0) AS with_body
		FROM hits %s`, where)

	var o overview
	row := s.conn.QueryRow(r.Context(), query, args...)
	if err := row.Scan(&o.TotalHits, &o.UniqueCountries, &o.UniquePaths, &o.UniqueAgents, &o.WithBody); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResponse(w, o)
}

// CSV export
func (s *Server) handleCSVExport(w http.ResponseWriter, r *http.Request) {
	f := parseFilters(r)
	where, args := buildWhereClause(f)

	query := fmt.Sprintf(`SELECT timestamp, method, path, user_agent, country, city,
	          content_type, body_size FROM hits %s ORDER BY timestamp DESC LIMIT 100000`, where)

	rows, err := s.conn.Query(r.Context(), query, args...)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=bot_hits.csv")

	writer := csv.NewWriter(w)
	writer.Write([]string{"Timestamp", "Method", "Path", "User Agent", "Country", "City", "Content Type", "Body Size"})

	for rows.Next() {
		var ts time.Time
		var method, path, ua, country, city, ct string
		var bodySize int64
		if err := rows.Scan(&ts, &method, &path, &ua, &country, &city, &ct, &bodySize); err != nil {
			continue
		}
		writer.Write([]string{
			ts.Format(time.RFC3339),
			method, path, ua, country, city, ct,
			strconv.FormatInt(bodySize, 10),
		})
	}
	writer.Flush()
}

// Available filter values
func (s *Server) handleFilters(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	type filters struct {
		Countries []string `json:"countries"`
		Methods   []string `json:"methods"`
	}
	var f filters

	rows, err := s.conn.Query(ctx, `SELECT DISTINCT country FROM hits ORDER BY country LIMIT 200`)
	if err == nil {
		for rows.Next() {
			var v string
			rows.Scan(&v)
			f.Countries = append(f.Countries, v)
		}
		rows.Close()
	}

	rows, err = s.conn.Query(ctx, `SELECT DISTINCT method FROM hits ORDER BY method`)
	if err == nil {
		for rows.Next() {
			var v string
			rows.Scan(&v)
			f.Methods = append(f.Methods, v)
		}
		rows.Close()
	}

	jsonResponse(w, f)
}

func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func intParam(r *http.Request, key string, def int) int {
	if v := r.URL.Query().Get(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
