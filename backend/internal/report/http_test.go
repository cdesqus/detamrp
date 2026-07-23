package report

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"order-stock/backend/internal/auth"
)

type routeTestAuthenticator struct{}

func (routeTestAuthenticator) Authenticate(context.Context, string) (auth.User, error) {
	return auth.User{}, nil
}

func TestRegisterRoutesExposesReceivingPDFAtPublicURL(t *testing.T) {
	router := gin.New()
	RegisterRoutes(router, nil, routeTestAuthenticator{})
	found := false
	for _, route := range router.Routes() {
		if route.Method == "GET" && route.Path == "/reports/receiving.pdf" {
			found = true
		}
	}
	if !found {
		t.Fatal("GET /reports/receiving.pdf is not registered")
	}
}
