package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Структура конфигурации (бери из yaml и/или env)
type Config struct {
	Server struct {
		Port string `yaml:"port"`
	} `yaml:"server"`

	Database struct {
		DSN string `yaml:"dsn"`
	} `yaml:"database"`

	JWTSecret      string `yaml:"jwt_secret"`
	GoogleClientID string `yaml:"google_client_id"`
	AuthToken      string `yaml:"auth_token"`
	BasicUser      string `yaml:"basic_user"`
	BasicPass      string `yaml:"basic_pass"`

	AI struct {
		Provider string `yaml:"provider"`
		APIKey   string `yaml:"api_key"`
	} `yaml:"ai"`

	// ...existing code...
}

// LoadConfig читает файл YAML (если существует) и потом переопределяет полями из окружения (env имеет приоритет).
func LoadConfig(path string) (*Config, error) {
	cfg := &Config{}

	// Попробовать прочитать файл, если он есть
	if data, err := os.ReadFile(path); err == nil {
		_ = yaml.Unmarshal(data, cfg) // ошибки парсинга вернём ниже, если нужно — можно вернуть
	}

	// Переопределить значениями из окружения (если заданы)
	if v := os.Getenv("PORT"); v != "" {
		cfg.Server.Port = v
	}
	if v := os.Getenv("DB_DSN"); v != "" {
		cfg.Database.DSN = v
	}
	if v := os.Getenv("JWT_SECRET"); v != "" {
		cfg.JWTSecret = v
	}
	if v := os.Getenv("GOOGLE_CLIENT_ID"); v != "" {
		cfg.GoogleClientID = v
	}
	if v := os.Getenv("AUTH_TOKEN"); v != "" {
		cfg.AuthToken = v
	}
	if v := os.Getenv("BASIC_USER"); v != "" {
		cfg.BasicUser = v
	}
	if v := os.Getenv("BASIC_PASS"); v != "" {
		cfg.BasicPass = v
	}

	return cfg, nil
}
