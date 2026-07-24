package dashboard

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"order-stock/backend/internal/auth"
)

type routeAuthenticator struct{}

func (routeAuthenticator) Authenticate(context.Context, string) (auth.User, error) {
	return auth.User{}, nil
}

func TestRegisterRoutesExposesDashboard(t *testing.T) {
	router := gin.New()
	RegisterRoutes(router, nil, routeAuthenticator{})
	for _, route := range router.Routes() {
		if route.Method == "GET" && route.Path == "/dashboard" {
			return
		}
	}
	t.Fatal("GET /dashboard is not registered")
}
