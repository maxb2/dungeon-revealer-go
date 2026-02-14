package config

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"os"
)

type Config struct {
	Port            int
	DataDir         string
	DMPassword      string
	PlayerPassword  string
	SessionSecret   string
}

func Parse() *Config {
	cfg := &Config{}

	flag.IntVar(&cfg.Port, "port", envInt("DR_PORT", 3000), "HTTP port")
	flag.StringVar(&cfg.DataDir, "data-dir", envStr("DR_DATA_DIR", "./data"), "Data directory path")
	flag.StringVar(&cfg.DMPassword, "dm-password", envStr("DR_DM_PASSWORD", ""), "DM password")
	flag.StringVar(&cfg.PlayerPassword, "player-password", envStr("DR_PLAYER_PASSWORD", ""), "Player password")
	flag.StringVar(&cfg.SessionSecret, "session-secret", envStr("DR_SESSION_SECRET", ""), "Session cookie secret")
	flag.Parse()

	if cfg.SessionSecret == "" {
		b := make([]byte, 32)
		rand.Read(b)
		cfg.SessionSecret = hex.EncodeToString(b)
	}

	return cfg
}

func envStr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n := 0
	for _, c := range v {
		if c < '0' || c > '9' {
			return fallback
		}
		n = n*10 + int(c-'0')
	}
	return n
}
