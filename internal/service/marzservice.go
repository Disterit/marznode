package service

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type MarznodeService struct {
	log *zap.SugaredLogger
}

func NewMarznodeService(log *zap.SugaredLogger) *MarznodeService {
	return &MarznodeService{log: log}
}

func (m *MarznodeService) ResolveTag(ctx *fiber.Ctx) error {
	return nil
}

func (m *MarznodeService) FetchUserStats(ctx *fiber.Ctx) error {
	return nil
}

func (m *MarznodeService) StreamBackendLogs(ctx *fiber.Ctx) error {
	return nil
}

func (m *MarznodeService) RestartBackend(ctx *fiber.Ctx) error {
	return nil
}

func (m *MarznodeService) GetBackendStats(ctx *fiber.Ctx) error {
	return nil
}
