package service

import (
	"go.uber.org/zap"
	"marznode/internal/repo"
)

type marznodeService struct {
	repo repo.MarznodeRepo
	log  *zap.SugaredLogger
}

func NewMarznodeService(repo repo.MarznodeRepo, log *zap.SugaredLogger) *marznodeService {
	return &marznodeService{
		repo: repo,
		log:  log,
	}
}
