package service

import (
	"context"
	"marznode/internal/repo"
	"marznode/pkg/backend/common/models"

	"go.uber.org/zap"
)

type marznodeService struct {
	repo repo.MarznodeRepo
	log  *zap.SugaredLogger
}

func NewMarznodeService(repo repo.MarznodeRepo, log *zap.SugaredLogger) MarznodeMemory {
	return &marznodeService{
		repo: repo,
		log:  log,
	}
}

// Inbound operations
func (s *marznodeService) ListInbounds(ctx context.Context, tags []string, includeUsers bool) ([]models.Inbound, error) {
	return s.repo.ListInbounds(ctx, tags, includeUsers)
}

func (s *marznodeService) GetInbound(ctx context.Context, tag string) (*models.Inbound, error) {
	return s.repo.GetInbound(ctx, tag)
}

func (s *marznodeService) RegisterInbound(ctx context.Context, inbound models.Inbound) error {
	return s.repo.RegisterInbound(ctx, inbound)
}

func (s *marznodeService) RemoveInbound(ctx context.Context, inbound models.Inbound) error {
	return s.repo.RemoveInbound(ctx, inbound)
}

func (s *marznodeService) RemoveInboundByTag(ctx context.Context, tag string) error {
	return s.repo.RemoveInboundByTag(ctx, tag)
}

// User operations
func (s *marznodeService) ListUsers(ctx context.Context) ([]models.User, error) {
	return s.repo.ListUsers(ctx)
}

func (s *marznodeService) GetUser(ctx context.Context, userID int64) (*models.User, error) {
	return s.repo.GetUser(ctx, userID)
}

func (s *marznodeService) ListInboundUsers(ctx context.Context, tag string) ([]models.User, error) {
	return s.repo.ListInboundUsers(ctx, tag)
}

func (s *marznodeService) AddUser(ctx context.Context, user models.User) error {
	return s.repo.AddUser(ctx, user)
}

func (s *marznodeService) RemoveUser(ctx context.Context, user models.User) error {
	return s.repo.RemoveUser(ctx, user)
}

func (s *marznodeService) UpdateUserInbounds(ctx context.Context, user models.User, inbounds []models.Inbound) error {
	return s.repo.UpdateUserInbounds(ctx, user, inbounds)
}

func (s *marznodeService) FlushUsers(ctx context.Context) error {
	return s.repo.FlushUsers(ctx)
}
