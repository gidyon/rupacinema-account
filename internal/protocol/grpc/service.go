package grpc

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/gidyon/rupacinema/account/pkg/api"
	"github.com/gidyon/rupacinema/account/pkg/config"
	"github.com/gidyon/rupacinema/notification/pkg/api"
	"google.golang.org/grpc"

	_ "github.com/go-sql-driver/mysql"

	"github.com/gidyon/rupacinema/account/internal/service"
)

// Opens a connection to mysql database
func createMySQLConn(cfg *config.Config) (*sql.DB, error) {
	// add MySQL driver specific parameter to parse date/time
	// Drop it for another database
	param := "parseTime=true"

	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?%s",
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBHost,
		cfg.DBSchema,
		param)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}
	return db, nil
}

// Creates the service
func createAccountAPIServer(
	ctx context.Context, cfg *config.Config,
) (account.AccountAPIServer, error) {
	// Create a *sql.DB instance
	db, err := createMySQLConn(cfg)
	if err != nil {
		return nil, err
	}

	// Remote services
	// Notification service
	notificationServiceConn, err := dialNotificationService(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to notification service: %v", err)
	}
	// Clear all connection when context cancel
	go func() {
		<-ctx.Done()
		notificationServiceConn.Close()
	}()

	return service.NewAccountAPIServer(
		ctx,
		db,
		notification.NewNotificationServiceClient(notificationServiceConn),
	)
}

// creates a connection to the notification service
func dialNotificationService(
	ctx context.Context, cfg *config.Config,
) (*grpc.ClientConn, error) {

	return grpc.DialContext(ctx, cfg.NotificationServiceAddress+cfg.NotificationServicePort, grpc.WithInsecure())
}
