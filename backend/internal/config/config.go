package config

import (
	"os"
)

type Config struct {
	// Server
	Port        string
	Environment string
	LogLevel    string

	// Database
	DatabaseURL string

	// OIDC
	OIDCIssuer       string
	OIDCClientID     string
	OIDCClientSecret string
	OIDCRedirectURI  string

	// JWT
	JWTSecret string
	JWTExpiry int

	// Kubernetes
	Kubeconfig   string
	K8SNamespace string

	// OpenCode
	OpenCodeInstallPath string

	// Config Encryption
	EncryptionKey string
}

func Load() *Config {
	return &Config{
		Port:                getEnv("PORT", "8080"),
		Environment:         getEnv("ENVIRONMENT", "development"),
		LogLevel:            getEnv("LOG_LEVEL", "info"),
		DatabaseURL:         getEnv("DATABASE_URL", "postgres://opencode:password@localhost:5432/opencode_dev"),
		OIDCIssuer:          getEnv("OIDC_ISSUER", ""),
		OIDCClientID:        getEnv("OIDC_CLIENT_ID", ""),
		OIDCClientSecret:    getEnv("OIDC_CLIENT_SECRET", ""),
		OIDCRedirectURI:     getEnv("OIDC_REDIRECT_URI", "http://localhost:5173/auth/callback"),
		JWTSecret:           getEnv("JWT_SECRET", ""),
		JWTExpiry:           3600,
		Kubeconfig:          getEnv("KUBECONFIG", ""),
		K8SNamespace:        getEnv("K8S_NAMESPACE", "opencode"),
		OpenCodeInstallPath: getEnv("OPENCODE_INSTALL_PATH", "/usr/local/bin/opencode"),
		EncryptionKey:       getEnv("CONFIG_ENCRYPTION_KEY", ""),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
