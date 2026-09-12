package model

import "time"

type Target struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
}

type ProbeLog struct {
	ID            int64     `json:"id"`
	TargetID      int64     `json:"target_id"`
	StatusCode    int       `json:"status_code"`
	IsUp          bool      `json:"is_up"`
	LatencyMs     int64     `json:"latency_ms"`
	SSLExpiryDays int       `json:"ssl_expiry_days"`
	ErrorMsg      string    `json:"error_msg,omitempty"`
	Headers       string    `json:"headers,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type TargetSummary struct {
	Target        Target    `json:"target"`
	LastProbe     *ProbeLog `json:"last_probe,omitempty"`
	UptimePercent float64   `json:"uptime_percent"`
	AvgLatencyMs  float64   `json:"avg_latency_ms"`
	TotalProbes   int       `json:"total_probes"`
}
