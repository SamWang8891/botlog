package geoip

import (
	"fmt"
	"net"

	"github.com/oschwald/geoip2-golang"
)

type Location struct {
	Country string
	City    string
}

type Resolver struct {
	db *geoip2.Reader
}

func New(path string) (*Resolver, error) {
	db, err := geoip2.Open(path)
	if err != nil {
		return nil, fmt.Errorf("geoip open %s: %w", path, err)
	}
	return &Resolver{db: db}, nil
}

func (r *Resolver) Lookup(ipStr string) Location {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return Location{Country: "Unknown", City: "Unknown"}
	}

	record, err := r.db.City(ip)
	if err != nil {
		return Location{Country: "Unknown", City: "Unknown"}
	}

	country := record.Country.Names["en"]
	city := record.City.Names["en"]

	if country == "" {
		country = "Unknown"
	}
	if city == "" {
		city = "Unknown"
	}

	return Location{Country: country, City: city}
}

func (r *Resolver) Close() {
	r.db.Close()
}
