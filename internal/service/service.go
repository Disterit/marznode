package service

import "github.com/gofiber/fiber/v2"

type Marznode interface {
	ResolveTag(ctx *fiber.Ctx) error
	FetchUserStats(ctx *fiber.Ctx) error
	StreamBackendLogs(ctx *fiber.Ctx) error
	RestartBackend(ctx *fiber.Ctx) error
	GetBackendStats(ctx *fiber.Ctx) error
}

type User interface {
	AddUser(ctx *fiber.Ctx) error
	UpdateUser(ctx *fiber.Ctx) error
	RepopulateUser(ctx *fiber.Ctx) error
	GetUserInfo(ctx *fiber.Ctx) error        // list_users
	GetByInbound(ctx *fiber.Ctx) error       // list_inbound_users
	RemoveUser(ctx *fiber.Ctx) error         // remove_user
	UpdateUserInbounds(ctx *fiber.Ctx) error // update_user_inbounds
	FlushUser(ctx *fiber.Ctx) error          // flush_users
}

type Inbound interface {
	GetAllInbounds(ctx *fiber.Ctx) error   // list_inbounds
	GetInboundsByTag(ctx *fiber.Ctx) error // list_inbounds
	RegisterInbound(ctx *fiber.Ctx) error  // register_inbound
	RemoveInbound(ctx *fiber.Ctx) error    // remove_inbound
}

type Service struct {
	MarzService Marznode
	User        User
	Inbound     Inbound
}

func NewService(marzService Marznode, user User, inbound Inbound) *Service {
	return &Service{
		MarzService: marzService,
		User:        user,
		Inbound:     inbound,
	}
}
