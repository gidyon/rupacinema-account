package main

import (
	"bufio"
	"context"
	"flag"
	"github.com/Sirupsen/logrus"
	"github.com/dgrijalva/jwt-go"
	http_server "github.com/gidyon/rupacinema/account/internal/protocol/http"
	"os"
	"strconv"

	"github.com/gidyon/rupacinema/account/pkg/config"
)

var (
	defaultLogLevel      = 0
	defaultLogTimeFormat = "2006-01-02T15:04:05Z07:00"
)

func main() {
	var (
		cfg      = &config.Config{}
		useFlags bool
	)

	flag.BoolVar(
		&useFlags, "uflag", false, "Whether to pass config in flags",
	)
	// gRPC section
	flag.StringVar(
		&cfg.GRPCPort, "grpc-port", ":5530", "gRPC port to bind",
	)
	// DB section
	flag.StringVar(
		&cfg.DBHost, "db-host", "mysqldb", "Database host",
	)
	flag.StringVar(
		&cfg.DBUser, "db-user", "root", "Database user",
	)
	flag.StringVar(
		&cfg.DBPassword, "db-password", "hakty11", "Database password",
	)
	// Logging section
	flag.IntVar(
		&cfg.LogLevel, "log-level", defaultLogLevel, "Global log level",
	)
	flag.StringVar(
		&cfg.LogTimeFormat, "log-time-format", defaultLogTimeFormat,
		"Print time format for logger e.g 2006-01-02T15:04:05Z07:00",
	)
	// JWT Section
	flag.StringVar(
		&cfg.JWTToken, "jwt-token", "", "Token to sign JWT claims",
	)
	// External Services
	flag.StringVar(
		&cfg.NotificationServiceAddress,
		"notification-host", "localhost",
		"Address of the notification service",
	)
	flag.StringVar(
		&cfg.NotificationServicePort,
		"notification-port", ":10039",
		"Port where the notification service is running in remote machine",
	)

	flag.Parse()

	if !useFlags {
		// Get from environmnent variables
		cfg = &config.Config{
			GRPCPort: os.Getenv("GRPC_PORT"),
			// Mysql section
			DBHost:                     os.Getenv("MYSQL_HOST"),
			DBUser:                     os.Getenv("MYSQL_USER"),
			DBPassword:                 os.Getenv("MYSQL_PASSWORD"),
			DBSchema:                   os.Getenv("MYSQL_DATABASE"),
			JWTToken:                   os.Getenv("JWT_SIGNING_TOKEN"),
			NotificationServiceAddress: os.Getenv("NOTIFICATION_ADDRESS"),
			NotificationServicePort:    os.Getenv("NOTIFICATION_PORT"),
		}
		logLevel := os.Getenv("LOG_LEVEL")
		logTimeFormat := os.Getenv("LOG_TIME_FORMAT")

		// Log Level
		if logLevel == "" {
			cfg.LogLevel = defaultLogLevel
		} else {
			logLevelInt64, err := strconv.ParseInt(logLevel, 10, 64)
			if err != nil {
				panic(err)
			}
			cfg.LogLevel = int(logLevelInt64)
		}

		// Log Time Format
		if logTimeFormat == "" {
			cfg.LogTimeFormat = defaultLogTimeFormat
		} else {
			cfg.LogTimeFormat = logTimeFormat
		}
	}

	cfg.JWTSigningMethod = jwt.SigningMethodHS256

	ctx, cancel := context.WithCancel(context.Background())

	s := bufio.NewScanner(os.Stdin)
	defer cancel()

	// Shutdown when user press q or Q
	go func() {
		for s.Scan() {
			if s.Text() == "q" || s.Text() == "Q" {
				cancel()
				return
			}
		}
	}()

	if err := http_server.Serve(ctx, cfg); err != nil {
		cancel()
		logrus.Fatalf("%v\n", err)
	}
}
