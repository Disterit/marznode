package cmd

import (
	"context"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"log"
	"marznode/api/pb"
	"marznode/internal/api"
	"marznode/internal/config"
	"marznode/internal/repo"
	"marznode/internal/service"
	"net"
	"os"
	"os/signal"
	"syscall"

	CustomLogger "marznode/internal/logger"
)

func main() {
	if err := godotenv.Load(config.EnvPath); err != nil {
		log.Fatal("Error loading .env file", zap.Error(err))
	}

	var cfg config.AppConfig
	if err := envconfig.Process("", &cfg); err != nil {
		log.Fatal("Error processing config", zap.Error(err))
	}

	logger, err := CustomLogger.NewLogger(cfg.LogLevel)
	if err != nil {
		log.Fatal("Error initializing logger", zap.Error(err))
	}

	pool, err := repo.Connection(context.Background(), cfg.PostgresDB)
	if err != nil {
		logger.Error("Error connecting to postgres", zap.Error(err))
	}

	if err := repo.CheckConnection(pool, logger); err != nil {
		logger.Error("Error to check connection to postgres", zap.Error(err))
	}

	marznodeRepository := repo.NewMarznodeRepository(pool, logger)

	repos := repo.NewRepository(marznodeRepository)

	marznodeService := service.NewMarznodeService(repos.MarznodeRepo, logger)

	services := service.NewService(marznodeService)

	handler := api.NewMarznodeHandler(services.MarzService, logger)

	server := grpc.NewServer()

	pb.RegisterMarzServiceServer(server, handler)

	lis, err := net.Listen("tcp", cfg.Grpc.Port)
	if err != nil {
		logger.Fatal("Failed to listen", zap.Error(err))
	}

	go func() {
		logger.Info("starting gRPC server")
		if err := server.Serve(lis); err != nil {
			logger.Fatal("Failed to serve", zap.Error(err))
		}
	}()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	<-signalChan

	logger.Info("Shutting down server...")

	server.GracefulStop()

	if err = repo.CloseConnection(pool); err != nil {
		logger.Error("Error closing connection", zap.Error(err))
	}

	logger.Info("Server stopped")
}
