package interceptor

import (
	"context"
	"fmt"
	"strings"

	"github.com/aleksiaichuk-innowise/inno_taxi/shared/consts"
	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type contextKey string

const (
	UserIDKey contextKey = "userID"
	RolesKey  contextKey = "roles"
)

type CustomClaims struct {
	Roles []string `json:"roles"`
	jwt.RegisteredClaims
}

func UserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDKey).(string)
	return userID, ok
}

func RolesFromContext(ctx context.Context) ([]string, bool) {
	roles, ok := ctx.Value(RolesKey).([]string)
	return roles, ok
}

func AuthInterceptor(secret string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "metadata is not provided")
		}

		authHeaders := md.Get(strings.ToLower(consts.JWTHeaderKey))
		if len(authHeaders) == 0 {
			return nil, status.Error(codes.Unauthenticated, "authorization header is missing")
		}

		if !strings.HasPrefix(authHeaders[0], consts.BearerPrefix) {
			return nil, status.Error(codes.Unauthenticated, "invalid authorization format")
		}
		tokenString := strings.TrimPrefix(authHeaders[0], consts.BearerPrefix)

		claims := &CustomClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		ctx = context.WithValue(ctx, UserIDKey, claims.Subject)
		ctx = context.WithValue(ctx, RolesKey, claims.Roles)

		return handler(ctx, req)
	}
}
