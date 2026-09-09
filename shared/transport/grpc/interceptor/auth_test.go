package interceptor

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const testSecret = "test-secret"

func signToken(t *testing.T, secret string, claims *CustomClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

func callWithAuthHeader(t *testing.T, authHeader string) (any, error) {
	t.Helper()
	ctx := context.Background()
	if authHeader != "" {
		ctx = metadata.NewIncomingContext(ctx, metadata.Pairs("authorization", authHeader))
	}

	var gotUserID string
	var gotRoles []string
	handler := func(ctx context.Context, req any) (any, error) {
		gotUserID, _ = UserIDFromContext(ctx)
		gotRoles, _ = RolesFromContext(ctx)
		return "ok", nil
	}

	resp, err := AuthInterceptor(testSecret)(ctx, nil, &grpc.UnaryServerInfo{}, handler)
	if err == nil {
		return struct {
			userID string
			roles  []string
		}{gotUserID, gotRoles}, nil
	}
	return resp, err
}

func TestAuthInterceptor_ValidToken(t *testing.T) {
	token := signToken(t, testSecret, &CustomClaims{
		Roles: []string{"user"},
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-1",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})

	res, err := callWithAuthHeader(t, "Bearer "+token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, ok := res.(struct {
		userID string
		roles  []string
	})
	if !ok {
		t.Fatalf("unexpected handler result type: %T", res)
	}
	if got.userID != "user-1" {
		t.Errorf("userID = %q, want %q", got.userID, "user-1")
	}
	if len(got.roles) != 1 || got.roles[0] != "user" {
		t.Errorf("roles = %v, want [user]", got.roles)
	}
}

func TestAuthInterceptor_MissingMetadata(t *testing.T) {
	_, err := callWithAuthHeader(t, "")
	assertUnauthenticated(t, err)
}

func TestAuthInterceptor_MalformedHeader(t *testing.T) {
	_, err := callWithAuthHeader(t, "Token abc123")
	assertUnauthenticated(t, err)
}

func TestAuthInterceptor_InvalidSignature(t *testing.T) {
	token := signToken(t, "wrong-secret", &CustomClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-1",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})

	_, err := callWithAuthHeader(t, "Bearer "+token)
	assertUnauthenticated(t, err)
}

func TestAuthInterceptor_ExpiredToken(t *testing.T) {
	token := signToken(t, testSecret, &CustomClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-1",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		},
	})

	_, err := callWithAuthHeader(t, "Bearer "+token)
	assertUnauthenticated(t, err)
}

func assertUnauthenticated(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Unauthenticated {
		t.Fatalf("expected codes.Unauthenticated, got %v", err)
	}
}
