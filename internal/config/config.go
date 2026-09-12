package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port              string
	DBPath            string
	ProbeInterval     time.Duration
	DefaultTargetURL  string
	DefaultTargetName string
}

func Load() *Config {
	port := getEnv("PORT", "8080")
	dbPath := getEnv("DB_PATH", "data/pulseops.db")

	intervalSec := 30
	if v := os.Getenv("PROBE_INTERVAL_SECONDS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			intervalSec = parsed
		}
	}

	return &Config{
		Port:              port,
		DBPath:            dbPath,
		ProbeInterval:     time.Duration(intervalSec) * time.Second,
		DefaultTargetURL:  getEnv("DEFAULT_TARGET_URL", "https://portfolioyusufjaelani.vercel.app"),
		DefaultTargetName: getEnv("DEFAULT_TARGET_NAME", "Yusuf Portfolio"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
