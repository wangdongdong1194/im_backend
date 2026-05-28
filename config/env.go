package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPPort         string
	GinMode          string
	SocketCORSOrigin string
	SocketSendRate   int
	SocketSendBurst  int
	MySQLDSN         string
	RedisAddr        string
	RedisUsername    string
	RedisPassword    string
	RedisDB          int
	RedisKeyPrefix   string
}

func Load() Config {
	loadEnvFiles()

	return Config{
		HTTPPort:         getEnv("HTTP_PORT", "3003"),
		GinMode:          getEnv("GIN_MODE", "debug"),
		SocketCORSOrigin: getEnv("SOCKET_CORS_ORIGIN", "*"),
		SocketSendRate:   getEnvAsInt("SOCKET_SEND_RATE", 40),
		SocketSendBurst:  getEnvAsInt("SOCKET_SEND_BURST", 80),
		MySQLDSN:         getEnv("MYSQL_DSN", ""),
		RedisAddr:        getEnv("REDIS_ADDR", "127.0.0.1:6379"),
		RedisUsername:    getEnv("REDIS_USERNAME", ""),
		RedisPassword:    getEnv("REDIS_PASSWORD", ""),
		RedisDB:          getEnvAsInt("REDIS_DB", 0),
		RedisKeyPrefix:   getEnv("REDIS_KEY_PREFIX", "im_backend"),
	}
}

func loadEnvFiles() {
	envPaths := make([]string, 0, 2)

	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		envPaths = append(envPaths, filepath.Join(execDir, ".env"))
	}

	envPaths = append(envPaths, ".env")

	for _, envPath := range envPaths {
		if _, err := os.Stat(envPath); err == nil {
			_ = godotenv.Overload(envPath)
			return
		}
	}
}

func (c Config) HTTPAddr() string {
	if strings.HasPrefix(c.HTTPPort, ":") {
		return c.HTTPPort
	}

	return fmt.Sprintf(":%s", c.HTTPPort)
}

func getEnv(key string, fallback string) string {
	value := strings.TrimSpace(strings.Trim(os.Getenv(key), "\"'"))
	if value == "" {
		return fallback
	}

	return value
}

func getEnvAsInt(key string, fallback int) int {
	value := getEnv(key, "")
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}
