package service

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
	"marznode/internal/repo"
)

type UserService struct {
	repos repo.User
	log   *zap.SugaredLogger
}

func NewUserService(repos repo.User, log *zap.SugaredLogger) *UserService {
	return &UserService{
		repos: repos,
		log:   log,
	}
}

func (u *UserService) AddUser(ctx *fiber.Ctx) error {
	return nil
}

func (u *UserService) UpdateUser(ctx *fiber.Ctx) error {
	return nil
}

func (u *UserService) RepopulateUser(ctx *fiber.Ctx) error {
	return nil
}

func (u *UserService) GetUserInfo(ctx *fiber.Ctx) error {
	return nil
}

func (u *UserService) GetByInbound(ctx *fiber.Ctx) error {
	return nil
}

func (u *UserService) RemoveUser(ctx *fiber.Ctx) error {
	return nil
}

func (u *UserService) UpdateUserInbounds(ctx *fiber.Ctx) error {
	return nil
}

func (u *UserService) FlushUser(ctx *fiber.Ctx) error {
	return nil
}
