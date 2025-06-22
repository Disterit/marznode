package pkg

import "marznode/internal/models"

type VPNBackend interface {
	Version() (string, error)
	Running() (bool, error)
	ContainsTag(tag string) (bool, error)
	Start(backendConfig any) error
	Restart(backendConfig any) error
	AddUser(user models.User, inbound models.Inbound) error
	RemoveUser(user models.User, inbound models.Inbound) error
	GetLogs() error
	GetUsage() error
	ListInbounds() error
	GetConfig() error
}
