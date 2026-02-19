package config

import (
	"os"
	"testing"
	"time"
)

func clearEnv() {
	vars := []string{
		"PORT", "ENVIRONMENT", "LOG_LEVEL",
		"GITHUB_WEBHOOK_SECRET",
		"MODEL_NAME", "MODEL_ENDPOINT", "MODEL_TIMEOUT", "MODEL_MAX_TOKENS",
		"AZURE_SUBSCRIPTION_ID", "AZURE_TENANT_ID", "AZURE_CLIENT_ID", "AZURE_CLIENT_SECRET",
		"TEAMS_WEBHOOK_URL", "SLACK_WEBHOOK_URL",
		"ENABLE_LLM", "ENABLE_NOTIFICATIONS",
	}
	for _, v := range vars {
		os.Unsetenv(v)
	}
}

func TestLoad_Defaults(t *testing.T) {
	clearEnv()

	cfg := Load()

	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want %q", cfg.Port, "8080")
	}
	if cfg.Environment != EnvDev {
		t.Errorf("Environment = %q, want %q", cfg.Environment, EnvDev)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
	}
	if cfg.ModelName != "gpt-4.1-mini" {
		t.Errorf("ModelName = %q, want %q", cfg.ModelName, "gpt-4.1-mini")
	}
	if cfg.ModelEndpoint != "https://models.inference.ai.azure.com" {
		t.Errorf("ModelEndpoint = %q, want default", cfg.ModelEndpoint)
	}
	if cfg.ModelTimeout != 30*time.Second {
		t.Errorf("ModelTimeout = %v, want 30s", cfg.ModelTimeout)
	}
	if cfg.ModelMaxTokens != 4096 {
		t.Errorf("ModelMaxTokens = %d, want 4096", cfg.ModelMaxTokens)
	}
	if !cfg.EnableLLM {
		t.Error("EnableLLM should default to true")
	}
	if cfg.EnableNotifications {
		t.Error("EnableNotifications should default to false in dev")
	}
}

func TestLoad_ProdEnvironment(t *testing.T) {
	clearEnv()
	os.Setenv("ENVIRONMENT", "prod")
	defer clearEnv()

	cfg := Load()

	if cfg.Environment != EnvProd {
		t.Errorf("Environment = %q, want %q", cfg.Environment, EnvProd)
	}
	if cfg.ModelName != "gpt-4.1" {
		t.Errorf("ModelName = %q, want %q", cfg.ModelName, "gpt-4.1")
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "info")
	}
	if !cfg.EnableNotifications {
		t.Error("EnableNotifications should default to true in prod")
	}
}

func TestLoad_TestEnvironment(t *testing.T) {
	clearEnv()
	os.Setenv("ENVIRONMENT", "test")
	defer clearEnv()

	cfg := Load()

	if cfg.Environment != EnvTest {
		t.Errorf("Environment = %q, want %q", cfg.Environment, EnvTest)
	}
	if cfg.ModelName != "gpt-4.1-mini" {
		t.Errorf("ModelName = %q, want %q", cfg.ModelName, "gpt-4.1-mini")
	}
}

func TestLoad_InvalidEnvironment(t *testing.T) {
	clearEnv()
	os.Setenv("ENVIRONMENT", "invalid")
	defer clearEnv()

	cfg := Load()

	if cfg.Environment != EnvDev {
		t.Errorf("Environment = %q, want %q (fallback)", cfg.Environment, EnvDev)
	}
}

func TestLoad_CustomValues(t *testing.T) {
	clearEnv()
	os.Setenv("PORT", "9090")
	os.Setenv("MODEL_TIMEOUT", "60s")
	os.Setenv("MODEL_MAX_TOKENS", "8192")
	os.Setenv("ENABLE_LLM", "false")
	defer clearEnv()

	cfg := Load()

	if cfg.Port != "9090" {
		t.Errorf("Port = %q, want %q", cfg.Port, "9090")
	}
	if cfg.ModelTimeout != 60*time.Second {
		t.Errorf("ModelTimeout = %v, want 60s", cfg.ModelTimeout)
	}
	if cfg.ModelMaxTokens != 8192 {
		t.Errorf("ModelMaxTokens = %d, want 8192", cfg.ModelMaxTokens)
	}
	if cfg.EnableLLM {
		t.Error("EnableLLM should be false")
	}
}

func TestValidate_DevNoSecret(t *testing.T) {
	clearEnv()
	cfg := Load()
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() should pass in dev without secret: %v", err)
	}
}

func TestValidate_ProdNoSecret(t *testing.T) {
	clearEnv()
	os.Setenv("ENVIRONMENT", "prod")
	defer clearEnv()

	cfg := Load()
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() should fail in prod without GITHUB_WEBHOOK_SECRET")
	}
}

func TestValidate_ProdWithSecret(t *testing.T) {
	clearEnv()
	os.Setenv("ENVIRONMENT", "prod")
	os.Setenv("GITHUB_WEBHOOK_SECRET", "secret123")
	defer clearEnv()

	cfg := Load()
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() should pass in prod with secret: %v", err)
	}
}

func TestEnvironmentHelpers(t *testing.T) {
	tests := []struct {
		env    Environment
		isProd bool
		isTest bool
		isDev  bool
	}{
		{EnvDev, false, false, true},
		{EnvTest, false, true, false},
		{EnvProd, true, false, false},
	}

	for _, tt := range tests {
		cfg := &Config{Environment: tt.env}
		if got := cfg.IsProd(); got != tt.isProd {
			t.Errorf("IsProd() with %s = %v, want %v", tt.env, got, tt.isProd)
		}
		if got := cfg.IsTest(); got != tt.isTest {
			t.Errorf("IsTest() with %s = %v, want %v", tt.env, got, tt.isTest)
		}
		if got := cfg.IsDev(); got != tt.isDev {
			t.Errorf("IsDev() with %s = %v, want %v", tt.env, got, tt.isDev)
		}
	}
}

func TestGetBoolEnv(t *testing.T) {
	tests := []struct {
		val      string
		defVal   bool
		expected bool
	}{
		{"true", false, true},
		{"1", false, true},
		{"yes", false, true},
		{"false", true, false},
		{"0", true, false},
		{"no", true, false},
		{"", true, true},
		{"", false, false},
		{"invalid", true, true},
	}

	for _, tt := range tests {
		os.Setenv("TEST_BOOL", tt.val)
		got := getBoolEnv("TEST_BOOL", tt.defVal)
		if got != tt.expected {
			t.Errorf("getBoolEnv(%q, %v) = %v, want %v", tt.val, tt.defVal, got, tt.expected)
		}
	}
	os.Unsetenv("TEST_BOOL")
}

func TestGetDurationEnv(t *testing.T) {
	os.Setenv("TEST_DUR", "5m")
	defer os.Unsetenv("TEST_DUR")

	got := getDurationEnv("TEST_DUR", time.Second)
	if got != 5*time.Minute {
		t.Errorf("getDurationEnv('5m') = %v, want 5m", got)
	}

	os.Setenv("TEST_DUR", "invalid")
	got = getDurationEnv("TEST_DUR", 10*time.Second)
	if got != 10*time.Second {
		t.Errorf("getDurationEnv('invalid') = %v, want 10s", got)
	}
}
