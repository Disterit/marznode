package repo

import (
	"context"
	"marznode/pkg/backend/common/models"
)

type MarznodeRepo interface {
	// Inbound operations
	ListInbounds(ctx context.Context, tags []string, includeUsers bool) ([]models.Inbound, error)
	GetInbound(ctx context.Context, tag string) (*models.Inbound, error)
	RegisterInbound(ctx context.Context, inbound models.Inbound) error
	RemoveInbound(ctx context.Context, inbound models.Inbound) error
	RemoveInboundByTag(ctx context.Context, tag string) error

	// User operations
	ListUsers(ctx context.Context) ([]models.User, error)
	GetUser(ctx context.Context, userID int64) (*models.User, error)
	ListInboundUsers(ctx context.Context, tag string) ([]models.User, error)
	AddUser(ctx context.Context, user models.User) error
	RemoveUser(ctx context.Context, user models.User) error
	UpdateUserInbounds(ctx context.Context, user models.User, inbounds []models.Inbound) error
	FlushUsers(ctx context.Context) error
}

type Repository struct {
	MarznodeRepo MarznodeRepo
}

func NewRepository(marznodeRepo MarznodeRepo) *Repository {
	return &Repository{
		MarznodeRepo: marznodeRepo,
	}
}
