package repo

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type marznodeRepository struct {
	pool *pgxpool.Pool
	log  *zap.SugaredLogger
}

func NewMarznodeRepository(
	pool *pgxpool.Pool,
	log *zap.SugaredLogger) MarznodeRepo {
	return &marznodeRepository{
		pool: pool,
		log:  log,
	}
}
