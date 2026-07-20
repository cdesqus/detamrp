package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func NewServer() http.Handler {
	router := gin.New()
	router.Use(gin.Recovery())
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	return router
}
