package repo

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"marznode/internal/config"
	"time"
)

func Connection(ctx context.Context, cfg config.PostgresDB) (*pgxpool.Pool, error) {

	connString := fmt.Sprintf("host=%s port=%d password=%s user=%s dbname=%s sslmode=%s pool_max_conns=%d "+
		"pool_max_conn_lifetime=%s pool_max_conn_idle_time=%s",
		cfg.Host,
		cfg.Port,
		cfg.Password,
		cfg.User,
		cfg.Database,
		cfg.SSLMode,
		cfg.PoolMaxConn,
		cfg.PoolMaxConnLifeTime,
		cfg.PoolMaxConnIdleTime,
	)

	configs, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing config")
	}

	pool, err := pgxpool.NewWithConfig(ctx, configs)
	if err != nil {
		return nil, errors.Wrap(err, "error connecting to database")
	}

	return pool, nil
}

func CheckConnection(pool *pgxpool.Pool, logger *zap.SugaredLogger) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := pool.Ping(ctx)
	if err != nil {
		return errors.Wrap(err, "database ping failed")
	}
	logger.Info("Database connection is successful")
	return nil
}

func CloseConnection(pool *pgxpool.Pool) error {
	if pool != nil {
		pool.Close()
		return nil
	}
	return errors.New("database connection is nil")
}
