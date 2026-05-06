package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	logingrpc "ride-sharing/services/login-service/internal/infrastructure/grpc"
	"ride-sharing/services/login-service/internal/infrastructure/repository"
	"ride-sharing/services/login-service/internal/service"
	"ride-sharing/services/login-service/pkg/migrate"
	"ride-sharing/services/login-service/pkg/token"
	"ride-sharing/shared/db"
	"ride-sharing/shared/env"
	"ride-sharing/shared/tracing"
	"syscall"
	"time"

	"google.golang.org/grpc"
)

const (
	HttpAddr = ":8085"
	GrpcAddr = ":9095"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize Tracing
	tracerCfg := tracing.Config{
		ServiceName:    "login-service",
		Environment:    env.GetString("ENVIRONMENT", "development"),
		JaegerEndpoint: env.GetString("JAEGER_ENDPOINT", "http://jaeger:14268/api/traces"),
	}

	sh, err := tracing.InitTracer(tracerCfg)
	if err != nil {
		log.Fatalf("Failed to initialize the tracer: %v", err)
	}
	defer sh(ctx)

	// Initialize PostgreSQL
	pgDB, err := db.NewPostgresDB(ctx, db.NewPostgresDefaultConfig())
	if err != nil {
		log.Fatalf("Failed to initialize PostgreSQL: %v", err)
	}
	defer pgDB.Close()

	// Run migrations
	migrationsDir := env.GetString("MIGRATIONS_DIR", "./services/login-service/migrations")
	if err := migrate.RunMigrations(pgDB, migrationsDir); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize dependencies
	userRepo := repository.NewPostgresUserRepository(pgDB)

	jwtSecret := env.GetString("JWT_SECRET", "change-me-in-production")
	tokenManager := token.NewManager(
		jwtSecret,
		15*time.Minute, // access token expiry
		7*24*time.Hour, // refresh token expiry
	)

	authService := service.NewAuthService(userRepo, tokenManager)

	// Signal handling
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		sig := <-sigCh
		log.Printf("Received signal: %v, shutting down...", sig)
		cancel()
	}()

	// HTTP health check server
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	httpServer := &http.Server{
		Addr:    HttpAddr,
		Handler: mux,
	}

	// gRPC server
	lis, err := net.Listen("tcp", GrpcAddr)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer(tracing.WithTracingInterceptors()...)
	logingrpc.NewGRPCHandler(grpcServer, authService)

	serverErrors := make(chan error, 2)

	go func() {
		log.Printf("Starting gRPC login-service on %s", lis.Addr().String())
		if err := grpcServer.Serve(lis); err != nil {
			serverErrors <- err
		}
	}()

	go func() {
		log.Printf("HTTP server listening on %s", HttpAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrors <- err
		}
	}()

	select {
	case err := <-serverErrors:
		log.Fatalf("Server error: %v", err)
	case <-ctx.Done():
		log.Println("Shutting down gracefully...")

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		grpcServer.GracefulStop()
		httpServer.Shutdown(shutdownCtx)
	}
}
