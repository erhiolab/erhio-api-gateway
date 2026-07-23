package models

import "time"

// APIKeyInfo API 密钥
type APIKeyInfo struct {
	ID                int64      `json:"id" db:"id"`
	UserID            int64      `json:"user_id" db:"user_id"`
	SecretID          string     `json:"secret_id" db:"secret_id"`
	SecretKey         string     `json:"secret_key" db:"secret_key"`
	ServiceID         int64      `json:"service_id" db:"service_id"`
	RouteIDs          []int64    `json:"route_ids" db:"_"`
	QPS               int64      `json:"qps" db:"qps"`
	QPM               int64      `json:"qpm" db:"qpm"`
	IPFilterType      int        `json:"ip_filter_type" db:"ip_filter_type"`
	RawIPList         *string    `json:"-" db:"ip_list"`
	IPList            []string   `json:"ip_list" db:"-"`
	CountryFilterType int        `json:"country_filter_type" db:"country_filter_type"`
	RawCountryList    *string    `json:"-" db:"country_list"`
	CountryList       []string   `json:"country_list" db:"-"`
	DomainFilterType  int        `json:"domain_filter_type" db:"domain_filter_type"`
	RawDomainList     *string    `json:"-" db:"domain_list"`
	DomainList        []string   `json:"domain_list" db:"-"`
	Enabled           bool       `json:"enabled" db:"enabled"`
	Banned            int64      `json:"banned" db:"banned"`
	ExpiresAt         *time.Time `json:"expires_at" db:"expires_at"`
}
