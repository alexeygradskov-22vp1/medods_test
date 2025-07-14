package api

import (
	"context"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"log"
	"medods/api"
	token3 "medods/internal/domain/token"
	"medods/internal/service/token"
	"medods/internal/service/user"
	"time"
)

const (
	FiberContextKey string = "FIBER_CONTEXT"
	UserGuidKey     string = "USER_GUID"
	UserAgentKey    string = "USER_AGENT"
	IPKey           string = "IP_KEY"
)

func StartHttpServer(ts *token.TokenService, us *user.UserService) {
	apiHandler := NewApiHandler(ts, us)
	app := fiber.New(
		fiber.Config{
			// Prefork:      true,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  10 * time.Second,
		},
	)
	app.Use(sendFiberContext())
	userGroup := app.Group("user")
	userGroup.Use(authMiddleware(ts))
	app.Static("/swagger", "./swagger-ui")
	app.Static("/api", "./api")
	api.RegisterHandlers(app, api.NewStrictHandler(apiHandler, nil))
	address := fmt.Sprintf(":8080")
	log.Fatal(app.Listen(address))
}

func authMiddleware(tokenService *token.TokenService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := c.Get("Authorization")
		if token == "" {
			return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
		}
		dst := token3.AccessTokenPayload{}
		err := tokenService.Valid(token, &dst)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
		}
		if !tokenService.VerifyToken(c.Context(), token) {
			return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
		}
		c.Locals(UserGuidKey, dst.Subject)
		return c.Next()
	}
}

func sendFiberContext() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := context.WithValue(c.UserContext(), FiberContextKey, c)
		c.SetUserContext(ctx)
		c.Locals(IPKey, c.IP())
		return c.Next()
	}
}
