package config

import (
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"github.com/sunriseex/capitalflow/pkg/errors"
)

type Config struct {
	TelegramToken             string
	TelegramUserID            int64
	AppVersion                string
	DataPath                  string
	DepositsDataPath          string
	DatabaseURL               string
	APIAuthToken              string
	JWTSecret                 string
	AccessTokenTTL            time.Duration
	RefreshTokenTTL           time.Duration
	CORSAllowedOrigins        []string
	RateLimitRequests         int
	RateLimitWindow           time.Duration
	AuthRateLimitRequests     int
	AuthRateLimitWindow       time.Duration
	MutationRateLimitRequests int
	MutationRateLimitWindow   time.Duration
	LogLevel                  slog.Level
}

var AppConfig *Config

func Init() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return errors.NewConfigurationError("не удалось получить домашнюю директорию", err)
	}

	envPaths := []string{
		filepath.Join(home, "nixos", "scripts", "capitalflow", "configs", ".env"),
		"./configs/.env",
	}

	var loaded bool
	for _, envPath := range envPaths {
		if err := godotenv.Load(envPath); err == nil {
			loaded = true
			break
		}
	}

	if !loaded {
		slog.Debug("env file not found, using defaults")
	}

	dataPath, err := expandPath(getEnv("DATA_PATH", "~/.config/waybar/payments.json"))
	if err != nil {
		return errors.NewConfigurationError("ошибка расширения пути DATA_PATH", err)
	}

	depositsDataPath, err := expandPath(getEnv("DEPOSITS_DATA_PATH", "~/.config/waybar/deposits.json"))
	if err != nil {
		return errors.NewConfigurationError("ошибка расширения пути DEPOSITS_DATA_PATH", err)
	}

	logLevel := slog.LevelError
	if envLogLevel := os.Getenv("LOG_LEVEL"); envLogLevel != "" {
		switch envLogLevel {
		case "debug":
			logLevel = slog.LevelDebug
		case "info":
			logLevel = slog.LevelInfo
		case "warn":
			logLevel = slog.LevelWarn
		case "error":
			logLevel = slog.LevelError
		}
	}

	AppConfig = &Config{
		TelegramToken:    getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramUserID:   getEnvInt64("TELEGRAM_USER_ID", 0),
		AppVersion:       getEnv("APP_VERSION", "0.1.0-dev"),
		DataPath:         dataPath,
		DepositsDataPath: depositsDataPath,
		DatabaseURL:      getEnv("DATABASE_URL", "postgres://capitalflow:capitalflow@localhost:5432/capitalflow?sslmode=disable"),
		LogLevel:         logLevel,
		APIAuthToken:     getEnv("API_AUTH_TOKEN", ""),
		JWTSecret:        getEnv("JWT_SECRET", ""),
		AccessTokenTTL:   getEnvDuration("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL:  getEnvDuration("REFRESH_TOKEN_TTL", 30*24*time.Hour),
		CORSAllowedOrigins: getEnvList("CORS_ALLOWED_ORIGINS", []string{
			"http://localhost:5173",
			"http://127.0.0.1:5173",
		}),
		RateLimitRequests:         getEnvInt("RATE_LIMIT_REQUESTS", 120),
		RateLimitWindow:           getEnvDuration("RATE_LIMIT_WINDOW", time.Minute),
		AuthRateLimitRequests:     getEnvInt("AUTH_RATE_LIMIT_REQUESTS", 5),
		AuthRateLimitWindow:       getEnvDuration("AUTH_RATE_LIMIT_WINDOW", time.Minute),
		MutationRateLimitRequests: getEnvInt("MUTATION_RATE_LIMIT_REQUESTS", 60),
		MutationRateLimitWindow:   getEnvDuration("MUTATION_RATE_LIMIT_WINDOW", time.Minute),
	}

	initLogger(logLevel)

	slog.Debug("Конфигурация инициализирована",
		"data_path", dataPath,
		"deposit_path", depositsDataPath,
		"log_level", logLevel)

	return nil
}

func initLogger(level slog.Level) {

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler

	if level == slog.LevelDebug {

		handler = slog.NewTextHandler(os.Stderr, opts)

	} else {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	}
	slog.SetDefault(slog.New(handler))
}

func expandPath(path string) (string, error) {
	if path == "" {
		return "", errors.NewConfigurationError("путь не может быть пустым", nil)
	}

	if strings.HasPrefix(path, "~/") || path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", errors.NewConfigurationError("не удалось получить домашнюю директорию", err)
		}

		if path == "~" {
			return home, nil
		}
		return filepath.Join(home, path[2:]), nil
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", errors.NewConfigurationError("ошибка получения абсолютного пути", err)
	}
	return absPath, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvList(key string, defaultValue []string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}

	items := make([]string, 0)
	for part := range strings.SplitSeq(value, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			items = append(items, part)
		}
	}
	if len(items) == 0 {
		return defaultValue
	}
	return items
}
