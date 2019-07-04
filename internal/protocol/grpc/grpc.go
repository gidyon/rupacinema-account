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

	opts = append(opts,
		grpc_middleware.WithUnaryServerChain(
			chainUnaryInterceptors(
				unaryLoggerInterceptors,
				[]grpc.UnaryServerInterceptor{unaryAuthInterceptor},
				unaryRecoveryInterceptors,
			)...,
		),
		grpc_middleware.WithStreamServerChain(
			chainStreamInterceptors(
				streamLoggerInterceptors,
				[]grpc.StreamServerInterceptor{streamAuthInterceptor},
				streamRecoveryInterceptors,
			)...,
		),
	)

	s := grpc.NewServer(opts...)

	accountService, err := createAccountAPIServer(ctx, cfg)
	if err != nil {
		return nil, err
	}

	// Register the account service
	account.RegisterAccountAPIServer(s, accountService)

	// Register reflection service on gRPC server.
	reflection.Register(s)

	return s, nil
}

type grpcUnaryInterceptorsSlice []grpc.UnaryServerInterceptor

func chainUnaryInterceptors(
	unaryInterceptorsSlice ...grpcUnaryInterceptorsSlice,
) []grpc.UnaryServerInterceptor {
	unaryInterceptors := make([]grpc.UnaryServerInterceptor, 0, len(unaryInterceptorsSlice))

	for _, unaryInterceptorSlice := range unaryInterceptorsSlice {
		for _, unaryInterceptor := range unaryInterceptorSlice {
			unaryInterceptors = append(unaryInterceptors, unaryInterceptor)
		}
	}

	return unaryInterceptors
}

type grpcStreamInterceptorsSlice []grpc.StreamServerInterceptor

func chainStreamInterceptors(
	streamInterceptorsSlice ...grpcStreamInterceptorsSlice,
) []grpc.StreamServerInterceptor {
	streamInterceptors := make([]grpc.StreamServerInterceptor, 0, len(streamInterceptorsSlice))

	for _, streamInterceptorSlice := range streamInterceptorsSlice {
		for _, streamInterceptor := range streamInterceptorSlice {
			streamInterceptors = append(streamInterceptors, streamInterceptor)
		}
	}

	return streamInterceptors
}
