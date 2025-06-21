package repo

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type InboundRepository struct {
	db  *pgxpool.Pool
	log *zap.SugaredLogger
}

func NewInboundRepository(db *pgxpool.Pool, logger *zap.SugaredLogger) *InboundRepository {
	return &InboundRepository{
		db:  db,
		log: logger,
	}
}

func (i *InboundRepository) GetAllInbounds() error {
	return nil
}

func (i *InboundRepository) GetInboundsByTag() error {
	return nil
}

func (i *InboundRepository) RegisterInbound() error {
	return nil
}

func (i *InboundRepository) RemoveInbound() error {
	return nil
}
