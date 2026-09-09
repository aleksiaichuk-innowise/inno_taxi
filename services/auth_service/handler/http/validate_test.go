package http

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gateway_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/auth_service/entity/gateway"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/auth_service/service"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type fakeSessionStore struct {
	active map[string]bool
}

func newFakeSessionStore() *fakeSessionStore {
	return &fakeSessionStore{active: map[string]bool{}}
}

func (f *fakeSessionStore) SaveAccessSession(_ context.Context, sid string, _ time.Duration) error {
	f.active[sid] = true
	return nil
}
func (f *fakeSessionStore) SaveRefreshSession(_ context.Context, sid string, _ time.Duration) error {
	f.active[sid] = true
	return nil
}
func (f *fakeSessionStore) AccessSessionExists(_ context.Context, sid string) (bool, error) {
	return f.active[sid], nil
}
func (f *fakeSessionStore) RefreshSessionExists(_ context.Context, sid string) (bool, error) {
	return f.active[sid], nil
}
func (f *fakeSessionStore) DeleteAccessSession(_ context.Context, sid string) error {
	delete(f.active, sid)
	return nil
}
func (f *fakeSessionStore) DeleteRefreshSession(_ context.Context, sid string) error {
	delete(f.active, sid)
	return nil
}

type fakeUserServiceGateway struct {
	user gateway_dto.UserInfo
}

func (f *fakeUserServiceGateway) VerifyCredentials(_ context.Context, _, _ string) (gateway_dto.UserInfo, error) {
	return f.user, nil
}

func testRouter(t *testing.T, roles []string) *gin.Engine {
	t.Helper()
	svc := service.NewAuthService(newFakeSessionStore(), &fakeUserServiceGateway{user: gateway_dto.UserInfo{ID: "user-1", Roles: roles}}, "test-secret", time.Minute, time.Hour)
	h := NewHandler(svc)
	r := gin.New()
	r.POST("/login", h.Login)
	r.GET("/validate", h.Validate)
	return r
}

func login(t *testing.T, r *gin.Engine) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/login", jsonBody(t, loginReq{Login: "u", Password: "p"}))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("login failed: %d %s", w.Code, w.Body.String())
	}
	var pair tokenPairResp
	if err := json.Unmarshal(w.Body.Bytes(), &pair); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	return pair.AccessToken
}

func jsonBody(t *testing.T, v any) io.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return bytes.NewReader(b)
}

func TestValidateHandler_ReadsTokenFromHeader(t *testing.T) {
	r := testRouter(t, []string{"user"})
	access := login(t, r)

	req := httptest.NewRequest(http.MethodGet, "/validate", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d; body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	if got := w.Header().Get("X-User-Id"); got != "user-1" {
		t.Errorf("X-User-Id header = %q, want %q", got, "user-1")
	}
	if got := w.Header().Get("X-Roles"); got != "user" {
		t.Errorf("X-Roles header = %q, want %q", got, "user")
	}
}

func TestValidateHandler_MissingAuthorizationHeader(t *testing.T) {
	r := testRouter(t, []string{"user"})

	req := httptest.NewRequest(http.MethodGet, "/validate", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestValidateHandler_RequiredRoleHeader(t *testing.T) {
	r := testRouter(t, []string{"user"})
	access := login(t, r)

	req := httptest.NewRequest(http.MethodGet, "/validate", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("X-Required-Role", "admin")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("got status %d, want %d; body=%s", w.Code, http.StatusForbidden, w.Body.String())
	}
}
