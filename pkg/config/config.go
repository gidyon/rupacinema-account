package config

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"strings"
)

// Config contains configuration variables for service
type Config struct {
	// gRPC server start parameters section
	// gRPC is TCP port to listen by gRPC server
	GRPCPort string

	// DB  parameters section
	// DBHost is host of database
	DBHost string
	// DBUser is username to connect to database
	DBUser string
	// DBPassword password to connect to database
	DBPassword string
	// DBSchema is schema of database
	DBSchema string

	// Cache DB parameters section. Redis in this case
	// Redis URL
	RedisURL string
	// RedisHost is the host of redis db
	RedisHost string
	// RedisPort is port of cache db
	RedisPort string
	// RedisUser is username to connect to redis
	RedisUser string
	// RedisPassword is the password to connect to redis
	RedisPassword string

	// Logging section
	// LogLevel id global loge Level: Debug(-1), Info(0), Warn(1), Error(2), DPanic(3), Panic(4), Fatal(5)
	LogLevel int
	// LogTimeFormat id print time format for logger e.g 2006-01-02T15:04:05Z07:00
	LogTimeFormat string

	// Certificates and Key section
	// Path to Certificate
	TLSCertPath string
	TLSKeyPath  string

	// External services section
	// Notification service
	NotificationServiceAddress  string
	NotificationServicePort     string
	NotificationServiceCertPath string

	// JWT Token
	JWTToken string
	// Signing method
	JWTSigningMethod jwt.SigningMethod
}

// Parse validates configuration data
func (cfg *Config) Parse() error {
	if strings.Trim(cfg.GRPCPort, " ") == "" {
		return fmt.Errorf("TCP port for gRPC server is required")
	}
	if strings.Trim(cfg.DBHost, " ") == "" {
		return fmt.Errorf("Database host is required")
	}
	if strings.Trim(cfg.DBUser, " ") == "" {
		return fmt.Errorf("Database user is required")
	}
	if strings.Trim(cfg.DBSchema, " ") == "" {
		return fmt.Errorf("Database schema is required")
	}

	return nil
}
