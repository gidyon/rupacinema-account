package rest

import (
	"context"
	"crypto/tls"
	"github.com/gidyon/rupacinema/account/internal/protocol"
	"github.com/gidyon/rupacinema/account/pkg/api"
	"github.com/gidyon/rupacinema/account/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc/credentials"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"

	grpc_server "github.com/gidyon/rupacinema/account/internal/protocol/grpc"
	"github.com/gidyon/rupacinema/account/pkg/config"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
)

// createRESTMux creates a REST client for gRPC service. A reverse proxy.
func createRESTMux(
	ctx context.Context,
	cfg *config.Config,
	opts ...runtime.ServeMuxOption,
) (*runtime.ServeMux, error) {

	tlsConfig, err := protocol.ClientTLS()
	if err != nil {
		return nil, err
	}

	dopts := []grpc.DialOption{
		grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
	}

	// gwmux := runtime.NewServeMux()
	gwmux := runtime.NewServeMux(runtime.WithMarshalerOption(runtime.MIMEWildcard,
		&runtime.JSONPb{OrigName: true, EmitDefaults: true}))

	err = account.RegisterAccountAPIHandlerFromEndpoint(
		ctx,
		gwmux,
		cfg.GRPCPort,
		dopts,
	)
	if err != nil {
		return nil, err
	}

	return gwmux, nil
}

// Serve serves both GRPC and HTTP on the same port
func serve(
	ctx context.Context,
	cfg *config.Config,
	grpcServer *grpc.Server,
	httpMux *http.ServeMux,
) error {
	// the grpcHandlerFunc takes an grpc server and a http muxer and will
	// route the request to the right place at runtime.
	mergeHandler := grpcHandlerFunc(grpcServer, httpMux)

	// Http server tls config
	tlsConfig, err := protocol.HTTPServerTLS()
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:    cfg.GRPCPort,
		Handler: mergeHandler,
	}

	lis, err := net.Listen("tcp", cfg.GRPCPort)
	if err != nil {
		return err
	}

	logger.Log.Info("gRPC and REST account service running", zap.String("gRPC Port", cfg.GRPCPort))

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			logger.Log.Warn("shutting account service....")
			srv.Shutdown(ctx)

			<-ctx.Done()
		}
	}()

	return srv.Serve(tls.NewListener(lis, tlsConfig))
}

// Serve serves both GRPC and HTTP on the same port
func Serve(
	ctx context.Context,
	cfg *config.Config,
) error {
	// gRPC server
	gRPCServer, err := grpc_server.CreateGRPCServer(ctx, cfg)
	if err != nil {
		return err
	}

	// REST muxer
	restMux, err := createRESTMux(ctx, cfg)
	if err != nil {
		return err
	}

	// register root Http multiplexer (mux)
	mux := http.NewServeMux()

	// register the gateway mux onto the root path.
	mux.Handle("/", restMux)

	// Test endpoint
	mux.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("accounts will work!"))
	})

	return serve(ctx, cfg, gRPCServer, mux)
}

// grpcHandlerFunc returns an http.Handler that delegates to grpcServer on incoming gRPC
// connections or otherHandler otherwise. Copied from cockroachdb.
func grpcHandlerFunc(grpcServer *grpc.Server, otherHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO(tamird): point to merged gRPC code rather than a PR.
		// This is a partial recreation of gRPC's internal checks https://github.com/grpc/grpc-go/pull/514/files#diff-95e9a25b738459a2d3030e1e6fa2a718R61
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			otherHandler.ServeHTTP(w, r)
		}
	})
}
