package common

import (
	"context"

	"github.com/highlight-apps/node-backend/backend/common/models"
)

type VPNBackend interface {
	BackendType() string
	ConfigFormat() int
	Version() (string, error)
	Running() bool
	ContainsTag(tag string) bool
	Start(ctx context.Context, backendConfig any) error
	Restart(ctx context.Context, backendConfig any) error
	AddUser(ctx context.Context, user models.User, inbound models.Inbound) error
	RemoveUser(ctx context.Context, user models.User, inbound models.Inbound) error
	GetLogs(ctx context.Context, includeBuffer bool) (<-chan string, error)
	GetUsages(ctx context.Context) (any, error)
	ListInbounds(ctx context.Context) ([]models.Inbound, error)
	GetConfig(ctx context.Context) (any, error)
}
