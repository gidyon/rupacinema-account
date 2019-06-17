package grpc

import (
	"context"
	"fmt"
	"github.com/gidyon/rupacinema/account/internal/protocol"
	"github.com/gidyon/rupacinema/account/pkg/config"
	"github.com/grpc-ecosystem/go-grpc-middleware"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"

	"github.com/gidyon/rupacinema/account/internal/protocol/grpc/middleware"
	"github.com/gidyon/rupacinema/account/pkg/api"
	"github.com/gidyon/rupacinema/account/pkg/logger"
)

// CreateGRPCServer ...
func CreateGRPCServer(ctx context.Context, cfg *config.Config) (*grpc.Server, error) {

	tlsConfig, err := protocol.GRPCServerTLS()
	if err != nil {
		return nil, err
	}

	creds := credentials.NewTLS(tlsConfig)

	opts := []grpc.ServerOption{
		grpc.Creds(creds),
	}

	err = logger.Init(cfg.LogLevel, cfg.LogTimeFormat)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %v", err)
	}

	// add logging middleware
	unaryLoggerInterceptors, streamLoggerInterceptors := middleware.AddLogging(logger.Log)

	// add auth middleware
	unaryAuthInterceptor, streamAuthInterceptor := middleware.AddAuthentication(
		[]byte(cfg.JWTToken), cfg.JWTSigningMethod,
	)

	// add recovery from panic middleware
	unaryRecoveryInterceptors, streamRecoveryInterceptors := middleware.AddRecovery()

	chainUnaryInterceptors()
	opts = append(opts,
		grpc_middleware.WithUnaryServerChain(
			append(
				unaryLoggerInterceptors,
				append(
					unaryRecoveryInterceptors,
					unaryAuthInterceptor,
				)...,
			)...,
		),
		grpc_middleware.WithStreamServerChain(
			append(
				streamLoggerInterceptors,
				append(
					streamRecoveryInterceptors,
					streamAuthInterceptor,
				)...,
			)...,
		))

	s := grpc.NewServer(opts...)

	accountService, err := createAccountAPIServer(ctx, cfg)
	if err != nil {
		return nil, err
	}

	account.RegisterAccountAPIServer(s, accountService)

	// Register reflection service on gRPC server.
	reflection.Register(s)

	return s, nil
}

func chainUnaryInterceptors(
	unaryInterceptors ...grpc.UnaryServerInterceptor,
) []grpc.UnaryServerInterceptor {
	unaryInterceptorsSlice := make([]grpc.UnaryServerInterceptor, 0, len(unaryInterceptors))

	for _, unaryInterceptor := range unaryInterceptors {
		unaryInterceptorsSlice = append(unaryInterceptorsSlice, unaryInterceptor)
	}

	return unaryInterceptorsSlice
}

func chainStreamInterceptors(
	streamInterceptors ...grpc.StreamServerInterceptor,
) []grpc.StreamServerInterceptor {
	streamInterceptorsSlice := make([]grpc.StreamServerInterceptor, 0, len(streamInterceptors))

	for _, streamInterceptor := range streamInterceptors {
		streamInterceptorsSlice = append(streamInterceptorsSlice, streamInterceptor)
	}

	return streamInterceptorsSlice
}
