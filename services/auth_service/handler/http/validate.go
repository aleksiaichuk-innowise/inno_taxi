package http

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/aleksiaichuk-innowise/inno_taxi/services/auth_service/errorsx"
	errresp "github.com/aleksiaichuk-innowise/inno_taxi/shared/errorsx"
	"github.com/gin-gonic/gin"
)

type validateReq struct {
	AccessToken string `json:"access_token" binding:"required"`
}

type validateResp struct {
	UserID string   `json:"user_id"`
	Roles  []string `json:"roles"`
}

func (h *Handler) Validate(c *gin.Context) {
	var req validateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errresp.HttpErrResp{Message: err.Error()})
		return
	}

	claims, err := h.svc.Validate(c.Request.Context(), req.AccessToken)
	if err != nil {
		if errors.Is(err, errorsx.ErrInvalidToken) {
			c.JSON(http.StatusUnauthorized, errresp.HttpErrResp{Message: err.Error()})
			return
		}
		slog.Error("validate", "error", err)
		c.JSON(http.StatusInternalServerError, errresp.HttpErrResp{Message: errresp.ErrInternal.Error()})
		return
	}

	c.JSON(http.StatusOK, validateResp{UserID: claims.UserID, Roles: claims.Roles})
}
