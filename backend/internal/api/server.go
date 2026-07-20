package api

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"order-stock/backend/internal/auth"
)

type Authenticator interface {
	Login(ctx context.Context, username, password string) (auth.LoginResult, error)
	Authenticate(ctx context.Context, token string) (auth.User, error)
	Logout(ctx context.Context, token string) error
}

type serverConfig struct {
	authenticator Authenticator
	cookieSecure  bool
}

type ServerOption func(*serverConfig)

func WithAuthenticator(authenticator Authenticator) ServerOption {
	return func(config *serverConfig) { config.authenticator = authenticator }
}

func WithSecureCookies() ServerOption {
	return func(config *serverConfig) { config.cookieSecure = true }
}

func NewServer(options ...ServerOption) http.Handler {
	config := serverConfig{}
	for _, option := range options {
		option(&config)
	}
	router := gin.New()
	router.Use(gin.Recovery())
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	if config.authenticator != nil {
		registerAuthRoutes(router, config)
	}
	return router
}
