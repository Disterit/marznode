package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"marznode/internal/service"
)

type Routers struct {
	Service *service.Service
}

func NewRouters(r *Routers) *fiber.App {
	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowMethods:  "*",
		AllowHeaders:  "*",
		ExposeHeaders: "Link",
		MaxAge:        300,
	}))

	user := app.Group("/user")
	{
		user.All("", r.Service.User.AddUser)
		user.All("", r.Service.User.FlushUser)
		user.All("", r.Service.User.RepopulateUser)
		user.All("", r.Service.User.GetUserInfo)
		user.All("", r.Service.User.GetByInbound)
		user.All("", r.Service.User.RemoveUser)
		user.All("", r.Service.User.UpdateUserInbounds)
		user.All("", r.Service.User.FlushUser)
	}

	inbound := app.Group("/inbound")
	{
		inbound.All("", r.Service.Inbound.GetAllInbounds)
		inbound.All("", r.Service.Inbound.GetInboundsByTag)
		inbound.All("", r.Service.Inbound.RegisterInbound)
		inbound.All("", r.Service.Inbound.RemoveInbound)
	}

	marznode := app.Group("/marznode")
	{
		marznode.All("", r.Service.MarzService.ResolveTag)
		marznode.All("", r.Service.MarzService.FetchUserStats)
		marznode.All("", r.Service.MarzService.StreamBackendLogs)
		marznode.All("", r.Service.MarzService.RestartBackend)
		marznode.All("", r.Service.MarzService.GetBackendStats)
	}

	return app
}
