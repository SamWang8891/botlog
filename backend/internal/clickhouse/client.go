package clickhouse

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type Hit struct {
	Timestamp   time.Time
	Method      string
	Path        string
	UserAgent   string
	Country     string
	City        string
	ContentType string
	BodyPreview string
	BodySize    int64
	Headers     map[string]string
	StatusCode  int
}

type Client struct {
	conn          driver.Conn
	db            string
	buffer        []Hit
	mu            sync.Mutex
	batchSize     int
	flushInterval time.Duration
	done          chan struct{}
}

func New(addr, db string, batchSize int, flushIntervalMs int) (*Client, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: db,
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		DialTimeout:     5 * time.Second,
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
	})
	if err != nil {
		return nil, fmt.Errorf("clickhouse open: %w", err)
	}

	if err := conn.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("clickhouse ping: %w", err)
	}

	c := &Client{
		conn:          conn,
		db:            db,
		buffer:        make([]Hit, 0, batchSize),
		batchSize:     batchSize,
		flushInterval: time.Duration(flushIntervalMs) * time.Millisecond,
		done:          make(chan struct{}),
	}

	go c.flushLoop()
	return c, nil
}

func (c *Client) Insert(h Hit) {
	c.mu.Lock()
	c.buffer = append(c.buffer, h)
	shouldFlush := len(c.buffer) >= c.batchSize
	c.mu.Unlock()

	if shouldFlush {
		c.flush()
	}
}

func (c *Client) flush() {
	c.mu.Lock()
	if len(c.buffer) == 0 {
		c.mu.Unlock()
		return
	}
	batch := c.buffer
	c.buffer = make([]Hit, 0, c.batchSize)
	c.mu.Unlock()

	ctx := context.Background()
	b, err := c.conn.PrepareBatch(ctx, "INSERT INTO hits")
	if err != nil {
		log.Printf("ERROR prepare batch: %v", err)
		return
	}

	for _, h := range batch {
		err := b.Append(
			h.Timestamp,
			h.Method,
			h.Path,
			h.UserAgent,
			h.Country,
			h.City,
			h.ContentType,
			h.BodyPreview,
			h.BodySize,
			h.Headers,
		)
		if err != nil {
			log.Printf("ERROR append: %v", err)
		}
	}

	if err := b.Send(); err != nil {
		log.Printf("ERROR send batch (%d hits): %v", len(batch), err)
	} else {
		log.Printf("Flushed %d hits to ClickHouse", len(batch))
	}
}

func (c *Client) flushLoop() {
	ticker := time.NewTicker(c.flushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.flush()
		case <-c.done:
			c.flush()
			return
		}
	}
}

func (c *Client) Close() {
	close(c.done)
	c.conn.Close()
}

func (c *Client) Conn() driver.Conn {
	return c.conn
}
