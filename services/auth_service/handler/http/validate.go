package http

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/aleksiaichuk-innowise/inno_taxi/services/auth_service/errorsx"
	"github.com/aleksiaichuk-innowise/inno_taxi/shared/consts"
	errresp "github.com/aleksiaichuk-innowise/inno_taxi/shared/errorsx"
	"github.com/gin-gonic/gin"
)

// Validate is built for nginx's auth_request module: it reads the token
// from the Authorization header (a subrequest carries headers, not a body)
// and, on success, echoes X-User-Id/X-Roles response headers so the
// gateway's auth_request_set can forward them downstream. An optional
// X-Required-Role request header lets the gateway declare, per location,
// which role a route needs - see gateway_service/nginx.conf.
func (h *Handler) Validate(c *gin.Context) {
	authHeader := c.GetHeader(consts.JWTHeaderKey)
	if !strings.HasPrefix(authHeader, consts.BearerPrefix) {
		c.JSON(http.StatusUnauthorized, errresp.HttpErrResp{Message: errorsx.ErrInvalidToken.Error()})
		return
	}
	accessToken := strings.TrimPrefix(authHeader, consts.BearerPrefix)
	requiredRole := c.GetHeader("X-Required-Role")

	claims, err := h.svc.Validate(c.Request.Context(), accessToken, requiredRole)
	if err != nil {
		switch {
		case errors.Is(err, errorsx.ErrInvalidToken):
			c.JSON(http.StatusUnauthorized, errresp.HttpErrResp{Message: err.Error()})
		case errors.Is(err, errorsx.ErrForbidden):
			c.JSON(http.StatusForbidden, errresp.HttpErrResp{Message: err.Error()})
		default:
			slog.Error("validate", "error", err)
			c.JSON(http.StatusInternalServerError, errresp.HttpErrResp{Message: errresp.ErrInternal.Error()})
		}
		return
	}

	c.Header("X-User-Id", claims.UserID)
	c.Header("X-Roles", strings.Join(claims.Roles, ","))
	c.JSON(http.StatusOK, validateResp{UserID: claims.UserID, Roles: claims.Roles})
}

type validateResp struct {
	UserID string   `json:"user_id"`
	Roles  []string `json:"roles"`
}
