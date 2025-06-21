package repo

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type UserRepository struct {
	db  *pgxpool.Pool
	log *zap.SugaredLogger
}

func NewUserRepository(db *pgxpool.Pool, log *zap.SugaredLogger) *UserRepository {
	return &UserRepository{
		db:  db,
		log: log,
	}
}

func (u *UserRepository) GetUserInfo() error {
	return nil
}

func (u *UserRepository) GetByInbound() error {
	return nil
}

func (u *UserRepository) RemoveUser() error {
	return nil
}

func (u *UserRepository) UpdateUserInbounds() error {
	return nil
}

func (u *UserRepository) FlushUser() error {
	return nil
}
