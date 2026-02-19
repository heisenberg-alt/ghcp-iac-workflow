// Package config provides centralized configuration for the IaC governance platform.
// Configuration is loaded from environment variables with sensible defaults.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Environment represents the deployment environment.
type Environment string

const (
	EnvDev  Environment = "dev"
	EnvTest Environment = "test"
	EnvProd Environment = "prod"
)

// Config holds all application configuration.
type Config struct {
	// Server
	Port        string      `json:"port"`
	Environment Environment `json:"environment"`
	LogLevel    string      `json:"log_level"`

	// Auth
	WebhookSecret string `json:"-"` // never serialize

	// HTTP Server timeouts
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
	IdleTimeout  time.Duration `json:"idle_timeout"`

	// Agent dispatch timeout
	AgentTimeout time.Duration `json:"agent_timeout"`

	// Request body size limit (DoS protection)
	MaxBodySize int64 `json:"max_body_size"`

	// LLM / GitHub Models
	ModelName      string        `json:"model_name"`
	ModelEndpoint  string        `json:"model_endpoint"`
	ModelTimeout   time.Duration `json:"model_timeout"`
	ModelMaxTokens int           `json:"model_max_tokens"`

	// Azure
	AzureSubscriptionID string `json:"azure_subscription_id"`
	AzureTenantID       string `json:"azure_tenant_id"`
	AzureClientID       string `json:"-"`
	AzureClientSecret   string `json:"-"`

	// Notifications
	TeamsWebhookURL string `json:"-"`
	SlackWebhookURL string `json:"-"`

	// Feature flags
	EnableLLM           bool `json:"enable_llm"`
	EnableNotifications bool `json:"enable_notifications"`
}

// Load reads configuration from environment variables with defaults.
func Load() *Config {
	env := Environment(getEnv("ENVIRONMENT", "dev"))
	if env != EnvDev && env != EnvTest && env != EnvProd {
		env = EnvDev
	}

	return &Config{
		Port:        getEnv("PORT", "8080"),
		Environment: env,
		LogLevel:    getEnv("LOG_LEVEL", logLevelForEnv(env)),

		WebhookSecret: os.Getenv("GITHUB_WEBHOOK_SECRET"),

		// HTTP Server timeouts
		ReadTimeout:  getDurationEnv("HTTP_READ_TIMEOUT", 30*time.Second),
		WriteTimeout: getDurationEnv("HTTP_WRITE_TIMEOUT", 120*time.Second),
		IdleTimeout:  getDurationEnv("HTTP_IDLE_TIMEOUT", 300*time.Second),
		AgentTimeout: getDurationEnv("AGENT_TIMEOUT", 90*time.Second),
		MaxBodySize:  getInt64Env("MAX_BODY_SIZE", 1<<20), // 1MB default

		ModelName:      getEnv("MODEL_NAME", modelForEnv(env)),
		ModelEndpoint:  getEnv("MODEL_ENDPOINT", "https://models.inference.ai.azure.com"),
		ModelTimeout:   getDurationEnv("MODEL_TIMEOUT", 30*time.Second),
		ModelMaxTokens: getIntEnv("MODEL_MAX_TOKENS", 4096),

		AzureSubscriptionID: os.Getenv("AZURE_SUBSCRIPTION_ID"),
		AzureTenantID:       os.Getenv("AZURE_TENANT_ID"),
		AzureClientID:       os.Getenv("AZURE_CLIENT_ID"),
		AzureClientSecret:   os.Getenv("AZURE_CLIENT_SECRET"),

		TeamsWebhookURL: os.Getenv("TEAMS_WEBHOOK_URL"),
		SlackWebhookURL: os.Getenv("SLACK_WEBHOOK_URL"),

		EnableLLM:           getBoolEnv("ENABLE_LLM", true),
		EnableNotifications: getBoolEnv("ENABLE_NOTIFICATIONS", env == EnvProd),
	}
}

// Validate checks that required configuration is present.
func (c *Config) Validate() error {
	if c.Port == "" {
		return fmt.Errorf("PORT is required")
	}
	if c.Environment == EnvProd && c.WebhookSecret == "" {
		return fmt.Errorf("GITHUB_WEBHOOK_SECRET is required in production")
	}
	return nil
}

// IsProd returns true if running in production.
func (c *Config) IsProd() bool { return c.Environment == EnvProd }

// IsTest returns true if running in test environment.
func (c *Config) IsTest() bool { return c.Environment == EnvTest }

// IsDev returns true if running in dev environment.
func (c *Config) IsDev() bool { return c.Environment == EnvDev }

func modelForEnv(env Environment) string {
	switch env {
	case EnvProd:
		return "gpt-4.1"
	case EnvTest:
		return "gpt-4.1-mini"
	default:
		return "gpt-4.1-mini"
	}
}

func logLevelForEnv(env Environment) string {
	switch env {
	case EnvProd:
		return "info"
	case EnvTest:
		return "debug"
	default:
		return "debug"
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getIntEnv(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			return n
		}
	}
	return defaultVal
}

func getInt64Env(key string, defaultVal int64) int64 {
	if val := os.Getenv(key); val != "" {
		if n, err := strconv.ParseInt(val, 10, 64); err == nil {
			return n
		}
	}
	return defaultVal
}

func getBoolEnv(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		switch strings.ToLower(val) {
		case "true", "1", "yes":
			return true
		case "false", "0", "no":
			return false
		}
	}
	return defaultVal
}

func getDurationEnv(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return defaultVal
}
