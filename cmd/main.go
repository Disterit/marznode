package cmd

import (
	"context"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"go.uber.org/zap"
	"log"
	"marznode/internal/api"
	"marznode/internal/config"
	"marznode/internal/repo"
	"marznode/internal/service"
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

	userRepository := repo.NewUserRepository(pool, logger)
	inboundRepository := repo.NewInboundRepository(pool, logger)

	repos := repo.NewRepository(inboundRepository, userRepository)

	userService := service.NewUserService(repos.User, logger)
	inboundService := service.NewInboundService(repos.Inbound, logger)
	marznodeService := service.NewMarznodeService(logger)

	services := service.NewService(marznodeService, userService, inboundService)

	app := api.NewRouters(&api.Routers{Service: services})

	go func() {
		logger.Info("starting http server")
		if err := app.Listen(":8080"); err != nil {
			logger.Error("Error starting http server", zap.Error(err))
		}
	}()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	<-signalChan

	logger.Info("Shutting down server...")

	if err := app.Shutdown(); err != nil {
		logger.Error("Error shutting down server", zap.Error(err))
	}

	if err = repo.CloseConnection(pool); err != nil {
		logger.Error("Error closing connection", zap.Error(err))
	}

	logger.Info("Server stopped")
}
