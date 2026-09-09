package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/entity/service"
	http_handler "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/handler/http"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/service"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type fakeDriverRepository struct {
	driver *service_dto.Driver
}

func (f *fakeDriverRepository) CreateDriver(_ context.Context, dto *service_dto.CreateDriverInput) (service_dto.Driver, error) {
	return service_dto.Driver{UserID: dto.UserID, TaxiType: dto.TaxiType}, nil
}

func (f *fakeDriverRepository) FindByUserID(_ context.Context, id string) (service_dto.Driver, error) {
	return service_dto.Driver{ID: "driver-1", UserID: id, TaxiType: service_dto.TaxiTypeEconomy, Status: service_dto.StatusOffline}, nil
}

func (f *fakeDriverRepository) UpdateStatusByUserID(_ context.Context, _, _ string) error {
	return nil
}

func (f *fakeDriverRepository) UpdateTaxiTypeByUserID(_ context.Context, _, _ string) error {
	return nil
}

func (f *fakeDriverRepository) FindByStatus(_ context.Context, _ service_dto.Status) ([]service_dto.Driver, error) {
	return nil, nil
}

func testRouter(t *testing.T) *gin.Engine {
	t.Helper()
	svc := service.NewDriverService(&fakeDriverRepository{})
	h := http_handler.NewDriverHandler(svc, validator.New())
	r := gin.New()
	registerRoutes(r, h, "test-secret")
	return r
}

func signToken(t *testing.T, subject string) string {
	t.Helper()
	claims := jwt.RegisteredClaims{
		Subject:   subject,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return token
}

func TestProfileRoutes_RequireAuth(t *testing.T) {
	r := testRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/profile", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestProfileRoutes_IdentityComesFromToken_NotPath(t *testing.T) {
	r := testRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/profile", nil)
	req.Header.Set("Authorization", "Bearer "+signToken(t, "token-user"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d; body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	if want := `"user_id":"token-user"`; !strings.Contains(w.Body.String(), want) {
		t.Fatalf("expected response to use token subject, got body=%s", w.Body.String())
	}

	// There must be no /profile/:user_id route left to trust a path-supplied identity.
	reqOther := httptest.NewRequest(http.MethodGet, "/profile/someone-else", nil)
	reqOther.Header.Set("Authorization", "Bearer "+signToken(t, "token-user"))
	wOther := httptest.NewRecorder()
	r.ServeHTTP(wOther, reqOther)
	if wOther.Code != http.StatusNotFound {
		t.Fatalf("expected /profile/:user_id to no longer exist, got status %d", wOther.Code)
	}
}

func TestProfileStatusRoute_RequiresAuth(t *testing.T) {
	r := testRouter(t)

	req := httptest.NewRequest(http.MethodPatch, "/profile/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want %d", w.Code, http.StatusUnauthorized)
	}
}
