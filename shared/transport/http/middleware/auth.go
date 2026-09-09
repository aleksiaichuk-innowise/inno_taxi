package middleware

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/aleksiaichuk-innowise/inno_taxi/shared/consts"
	"github.com/aleksiaichuk-innowise/inno_taxi/shared/errorsx"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type CustomClaims struct {
	Roles []string `json:"roles"`
	jwt.RegisteredClaims
}

func Auth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader(consts.JWTHeaderKey)
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorsx.HttpErrResp{Message: "Authorization header is empty"})
			return
		}

		if !strings.HasPrefix(authHeader, consts.BearerPrefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorsx.HttpErrResp{Message: "Authorization header format is invalid"})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, consts.BearerPrefix)
		claims := &CustomClaims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorsx.HttpErrResp{Message: "Unauthorized"})
			return
		}

		c.Set("userID", claims.Subject)
		c.Set("roles", claims.Roles)

		c.Next()
	}
}

func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roles := c.GetStringSlice("roles")

		if ok := slices.Contains(roles, role); !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, errorsx.HttpErrResp{Message: "role not allowed"})
			return
		}
		c.Next()
	}
}
