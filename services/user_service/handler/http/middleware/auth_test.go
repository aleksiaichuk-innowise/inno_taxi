package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type protectedResp struct {
	UserID string   `json:"userID"`
	Roles  []string `json:"roles"`
}

func newAuthTestRouter(secret string) *gin.Engine {
	r := gin.New()
	r.Use(Auth(secret))
	r.GET("/protected", func(c *gin.Context) {
		roles, _ := c.Get("roles")
		rolesSlice, _ := roles.([]string)
		c.JSON(http.StatusOK, protectedResp{
			UserID: c.GetString("userID"),
			Roles:  rolesSlice,
		})
	})
	return r
}

func signToken(t *testing.T, secret string, claims *CustomClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

func doRequest(r *gin.Engine, authHeader string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestAuth_ValidToken(t *testing.T) {
	secret := "test-secret"
	r := newAuthTestRouter(secret)

	token := signToken(t, secret, &CustomClaims{
		Roles: []string{"user"},
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-1",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})

	w := doRequest(r, "Bearer "+token)

	if w.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d; body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	var got protectedResp
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.UserID != "user-1" {
		t.Errorf("got userID %q, want %q", got.UserID, "user-1")
	}
	if len(got.Roles) != 1 || got.Roles[0] != "user" {
		t.Errorf("got roles %v, want [user]", got.Roles)
	}
}

func TestAuth_MissingHeader(t *testing.T) {
	r := newAuthTestRouter("test-secret")

	w := doRequest(r, "")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("got status %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuth_MalformedHeader(t *testing.T) {
	r := newAuthTestRouter("test-secret")

	w := doRequest(r, "Token abc123")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("got status %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuth_InvalidSignature(t *testing.T) {
	secret := "test-secret"
	r := newAuthTestRouter(secret)

	token := signToken(t, "wrong-secret", &CustomClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-1",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})

	w := doRequest(r, "Bearer "+token)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("got status %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuth_ExpiredToken(t *testing.T) {
	secret := "test-secret"
	r := newAuthTestRouter(secret)

	token := signToken(t, secret, &CustomClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-1",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		},
	})

	w := doRequest(r, "Bearer "+token)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("got status %d, want %d", w.Code, http.StatusUnauthorized)
	}
}
