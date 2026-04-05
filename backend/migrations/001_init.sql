-- Create database
CREATE DATABASE IF NOT EXISTS botlog;

-- Raw hits table (MergeTree, ordered by time for fast range queries)
CREATE TABLE IF NOT EXISTS botlog.hits (
    timestamp    DateTime64(3, 'UTC'),
    method       LowCardinality(String),
    path         String,
    user_agent   String,
    country      LowCardinality(String),
    city         LowCardinality(String),
    content_type LowCardinality(String),
    body_preview String,
    body_size    Int64,
    headers      Map(String, String)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (timestamp, country, method)
TTL toDateTime(timestamp) + INTERVAL 100 YEAR
SETTINGS index_granularity = 8192;

-- Materialized view: hits per hour by country
CREATE MATERIALIZED VIEW IF NOT EXISTS botlog.hits_hourly_country
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(hour)
ORDER BY (hour, country)
AS SELECT
    toStartOfHour(timestamp) AS hour,
    country,
    count() AS hits
FROM botlog.hits
GROUP BY hour, country;

-- Materialized view: hits per hour by method
CREATE MATERIALIZED VIEW IF NOT EXISTS botlog.hits_hourly_method
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(hour)
ORDER BY (hour, method)
AS SELECT
    toStartOfHour(timestamp) AS hour,
    method,
    count() AS hits
FROM botlog.hits
GROUP BY hour, method;

-- Materialized view: hits per hour by path (top endpoints)
CREATE MATERIALIZED VIEW IF NOT EXISTS botlog.hits_hourly_path
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(hour)
ORDER BY (hour, path)
AS SELECT
    toStartOfHour(timestamp) AS hour,
    path,
    count() AS hits
FROM botlog.hits
GROUP BY hour, path;

-- Materialized view: hits per hour by user_agent
CREATE MATERIALIZED VIEW IF NOT EXISTS botlog.hits_hourly_agent
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(hour)
ORDER BY (hour, user_agent)
AS SELECT
    toStartOfHour(timestamp) AS hour,
    user_agent,
    count() AS hits
FROM botlog.hits
GROUP BY hour, user_agent;

-- Materialized view: daily summary
CREATE MATERIALIZED VIEW IF NOT EXISTS botlog.hits_daily
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(day)
ORDER BY (day, country, method)
AS SELECT
    toStartOfDay(timestamp) AS day,
    country,
    method,
    count() AS hits
FROM botlog.hits
GROUP BY day, country, method;
