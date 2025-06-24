package repo

import (
	"context"
	"fmt"
	"marznode/pkg/backend/common/models"
	"sync"

	"go.uber.org/zap"
)

// InMemoryStorage - аналог Python storage с inbounds и users
type InMemoryStorage struct {
	inbounds map[string]models.Inbound // tag -> Inbound
	users    map[int64]models.User     // user_id -> User (изменено на int64 как в Python)
	mutex    sync.RWMutex
}

type marznodeRepository struct {
	storage *InMemoryStorage
	log     *zap.SugaredLogger
}

func NewMarznodeRepository(log *zap.SugaredLogger) MarznodeRepo {
	return &marznodeRepository{
		storage: &InMemoryStorage{
			inbounds: make(map[string]models.Inbound),
			users:    make(map[int64]models.User),
		},
		log: log,
	}
}

// ListUsers - аналог Python list_users
func (r *marznodeRepository) ListUsers(ctx context.Context) ([]models.User, error) {
	r.storage.mutex.RLock()
	defer r.storage.mutex.RUnlock()

	users := make([]models.User, 0, len(r.storage.users))
	for _, user := range r.storage.users {
		users = append(users, user)
	}
	return users, nil
}

// GetUser - получение пользователя по ID (аналог Python list_users с user_id)
func (r *marznodeRepository) GetUser(ctx context.Context, userID int64) (*models.User, error) {
	r.storage.mutex.RLock()
	defer r.storage.mutex.RUnlock()

	if user, exists := r.storage.users[userID]; exists {
		return &user, nil
	}
	return nil, nil // В Python возвращает None если не найден
}

// ListInbounds - точный аналог Python list_inbounds
func (r *marznodeRepository) ListInbounds(ctx context.Context, tags []string, includeUsers bool) ([]models.Inbound, error) {
	r.storage.mutex.RLock()
	defer r.storage.mutex.RUnlock()

	if tags == nil {
		// Возвращаем все inbounds - аналог list(self.storage["inbounds"].values())
		inbounds := make([]models.Inbound, 0, len(r.storage.inbounds))
		for _, inbound := range r.storage.inbounds {
			inbounds = append(inbounds, inbound)
		}
		return inbounds, nil
	}

	// Если передан список tags - аналог list comprehension
	var inbounds []models.Inbound
	for _, tag := range tags {
		if inbound, exists := r.storage.inbounds[tag]; exists {
			inbounds = append(inbounds, inbound)
		}
	}
	return inbounds, nil
}

// GetInbound - получение одного inbound по tag (аналог одиночного tag в Python)
func (r *marznodeRepository) GetInbound(ctx context.Context, tag string) (*models.Inbound, error) {
	r.storage.mutex.RLock()
	defer r.storage.mutex.RUnlock()

	if inbound, exists := r.storage.inbounds[tag]; exists {
		return &inbound, nil
	}
	return nil, fmt.Errorf("inbound with tag %s not found", tag)
}

// ListInboundUsers - точный аналог Python list_inbound_users
func (r *marznodeRepository) ListInboundUsers(ctx context.Context, tag string) ([]models.User, error) {
	r.storage.mutex.RLock()
	defer r.storage.mutex.RUnlock()

	var users []models.User
	for _, user := range r.storage.users {
		// Проверяем есть ли у пользователя inbound с указанным tag
		for _, inbound := range user.Inbounds {
			if inbound.Tag == tag {
				users = append(users, user)
				break // break из внутреннего цикла как в Python
			}
		}
	}
	return users, nil
}

// AddUser - добавление пользователя
func (r *marznodeRepository) AddUser(ctx context.Context, user models.User) error {
	r.storage.mutex.Lock()
	defer r.storage.mutex.Unlock()

	r.storage.users[user.ID] = user
	r.log.Infof("Added user: %s (ID: %d)", user.Username, user.ID)
	return nil
}

// RemoveUser - точный аналог Python remove_user
func (r *marznodeRepository) RemoveUser(ctx context.Context, user models.User) error {
	r.storage.mutex.Lock()
	defer r.storage.mutex.Unlock()

	// del self.storage["users"][user.id]
	delete(r.storage.users, user.ID)
	r.log.Infof("Removed user: %s (ID: %d)", user.Username, user.ID)
	return nil
}

// UpdateUserInbounds - точный аналог Python update_user_inbounds
func (r *marznodeRepository) UpdateUserInbounds(ctx context.Context, user models.User, inbounds []models.Inbound) error {
	r.storage.mutex.Lock()
	defer r.storage.mutex.Unlock()

	// if self.storage["users"].get(user.id):
	if existingUser, exists := r.storage.users[user.ID]; exists {
		// self.storage["users"][user.id].inbounds = inbounds
		existingUser.Inbounds = inbounds
		r.storage.users[user.ID] = existingUser
	}
	// user.inbounds = inbounds
	// self.storage["users"][user.id] = user
	user.Inbounds = inbounds
	r.storage.users[user.ID] = user

	r.log.Infof("Updated inbounds for user: %s (ID: %d)", user.Username, user.ID)
	return nil
}

// RegisterInbound - точный аналог Python register_inbound
func (r *marznodeRepository) RegisterInbound(ctx context.Context, inbound models.Inbound) error {
	r.storage.mutex.Lock()
	defer r.storage.mutex.Unlock()

	// self.storage["inbounds"][inbound.tag] = inbound
	r.storage.inbounds[inbound.Tag] = inbound
	r.log.Infof("Registered inbound: %s", inbound.Tag)
	return nil
}

// RemoveInbound - точный аналог Python remove_inbound
func (r *marznodeRepository) RemoveInbound(ctx context.Context, inbound models.Inbound) error {
	return r.RemoveInboundByTag(ctx, inbound.Tag)
}

// RemoveInboundByTag - для случая когда передается строка tag
func (r *marznodeRepository) RemoveInboundByTag(ctx context.Context, tag string) error {
	r.storage.mutex.Lock()
	defer r.storage.mutex.Unlock()

	// tag = inbound if isinstance(inbound, str) else inbound.tag
	// if tag in self.storage["inbounds"]:
	//     self.storage["inbounds"].pop(tag)
	if _, exists := r.storage.inbounds[tag]; exists {
		delete(r.storage.inbounds, tag)
	}

	// for user_id, user in self.storage["users"].items():
	//     user.inbounds = list(filter(lambda inb: inb.tag != tag, user.inbounds))
	for userID, user := range r.storage.users {
		var filteredInbounds []models.Inbound
		for _, inb := range user.Inbounds {
			if inb.Tag != tag {
				filteredInbounds = append(filteredInbounds, inb)
			}
		}
		user.Inbounds = filteredInbounds
		r.storage.users[userID] = user
	}

	r.log.Infof("Removed inbound: %s", tag)
	return nil
}

// FlushUsers - точный аналог Python flush_users
func (r *marznodeRepository) FlushUsers(ctx context.Context) error {
	r.storage.mutex.Lock()
	defer r.storage.mutex.Unlock()

	// self.storage["users"] = {}
	r.storage.users = make(map[int64]models.User)
	r.log.Info("Flushed all users")
	return nil
}
