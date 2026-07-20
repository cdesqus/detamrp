package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"order-stock/backend/internal/auth"
)

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func registerAuthRoutes(router *gin.Engine, config serverConfig) {
	router.POST("/auth/login", func(c *gin.Context) {
		var request loginRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "username and password are required"})
			return
		}
		result, err := config.authenticator.Login(c.Request.Context(), request.Username, request.Password)
		if err != nil {
			if errors.Is(err, auth.ErrInvalidCredentials) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": auth.ErrInvalidCredentials.Error()})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "login could not be completed"})
			}
			return
		}
		http.SetCookie(c.Writer, &http.Cookie{Name: "session", Value: result.Token, Path: "/", HttpOnly: true, Secure: config.cookieSecure, SameSite: http.SameSiteLaxMode, Expires: result.ExpiresAt})
		c.JSON(http.StatusOK, gin.H{"user": gin.H{"id": result.User.ID, "username": result.User.Username, "displayName": result.User.DisplayName, "permissions": result.User.Permissions}})
	})
	router.GET("/auth/me", func(c *gin.Context) {
		cookie, err := c.Cookie("session")
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": auth.ErrUnauthenticated.Error()})
			return
		}
		user, err := config.authenticator.Authenticate(c.Request.Context(), cookie)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": auth.ErrUnauthenticated.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"user": gin.H{"id": user.ID, "username": user.Username, "displayName": user.DisplayName, "permissions": user.Permissions}})
	})
	router.POST("/auth/logout", func(c *gin.Context) {
		cookie, _ := c.Cookie("session")
		if err := config.authenticator.Logout(c.Request.Context(), cookie); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "logout could not be completed"})
			return
		}
		http.SetCookie(c.Writer, &http.Cookie{Name: "session", Value: "", Path: "/", HttpOnly: true, Secure: config.cookieSecure, SameSite: http.SameSiteLaxMode, MaxAge: -1})
		c.Status(http.StatusNoContent)
	})
}
