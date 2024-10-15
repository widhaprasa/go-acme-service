package middleware

import (
	"encoding/base64"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/widhaprasa/go-acme-service/env"
)

func AuthorizeHeader() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authHeader := ctx.GetHeader("Authorization")
		username := env.SERVICE_USERNAME
		password := env.SERVICE_PASSWORD
		if username == "" {
			return
		}
		auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))

		if authHeader != "Basic "+auth {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
	}
}
