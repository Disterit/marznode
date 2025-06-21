package service

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
	"marznode/internal/repo"
)

type InboundService struct {
	repos repo.Inbound
	log   *zap.SugaredLogger
}

func NewInboundService(repos repo.Inbound, log *zap.SugaredLogger) *InboundService {
	return &InboundService{
		repos: repos,
		log:   log,
	}
}

func (s *InboundService) GetAllInbounds(ctx *fiber.Ctx) error {
	return nil
}

func (s *InboundService) GetInboundsByTag(ctx *fiber.Ctx) error {
	return nil
}

func (s *InboundService) RegisterInbound(ctx *fiber.Ctx) error {
	return nil
}

func (s *InboundService) RemoveInbound(ctx *fiber.Ctx) error {
	return nil
}
