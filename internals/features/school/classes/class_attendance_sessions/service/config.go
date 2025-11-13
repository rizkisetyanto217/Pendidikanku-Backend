package service

// import (
// 	"fmt"
// 	"os"
// 	"strconv"
// 	"time"
// )

// type Config struct {
// 	Enabled    bool
// 	WindowMins int
// 	PollEvery  time.Duration
// 	BatchSize  int
// 	AutoOpen   bool
// }

// func getenvBool(key string, def bool) bool {
// 	v := os.Getenv(key)
// 	switch v {
// 	case "1", "true", "TRUE", "on", "ON", "yes", "YES":
// 		return true
// 	case "0", "false", "FALSE", "off", "OFF", "no", "NO":
// 		return false
// 	}
// 	return def
// }

// func getenvInt(key string, def int) int {
// 	if s := os.Getenv(key); s != "" {
// 		if n, err := strconv.Atoi(s); err == nil {
// 			return n
// 		}
// 	}
// 	return def
// }

// func getenvDur(key string, def time.Duration) time.Duration {
// 	if s := os.Getenv(key); s != "" {
// 		if d, err := time.ParseDuration(s); err == nil {
// 			return d
// 		}
// 	}
// 	return def
// }

// func LoadConfig() (Config, error) {
// 	cfg := Config{
// 		Enabled:    getenvBool("ATTENDANCE_AUTO_SEED", true),
// 		WindowMins: getenvInt("ATTENDANCE_SEED_WINDOW_MIN", 60),
// 		PollEvery:  getenvDur("ATTENDANCE_SEED_POLL_EVERY", 1*time.Minute),
// 		BatchSize:  getenvInt("ATTENDANCE_SEED_BATCH", 300),
// 		AutoOpen:   true, // set status 'open' saat seed
// 	}

// 	if cfg.WindowMins <= 0 {
// 		return cfg, fmt.Errorf("ATTENDANCE_SEED_WINDOW_MIN must be > 0")
// 	}
// 	if cfg.BatchSize <= 0 {
// 		cfg.BatchSize = 300
// 	}
// 	if cfg.PollEvery < 15*time.Second {
// 		cfg.PollEvery = time.Minute
// 	}

// 	return cfg, nil
// }
