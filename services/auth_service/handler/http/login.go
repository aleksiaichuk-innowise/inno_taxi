package http

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/aleksiaichuk-innowise/inno_taxi/services/auth_service/errorsx"
	errresp "github.com/aleksiaichuk-innowise/inno_taxi/shared/errorsx"
	"github.com/gin-gonic/gin"
)

type loginReq struct {
	Login    string `json:"login" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type tokenPairResp struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errresp.HttpErrResp{Message: err.Error()})
		return
	}

	pair, err := h.svc.Login(c.Request.Context(), req.Login, req.Password)
	if err != nil {
		if errors.Is(err, errorsx.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, errresp.HttpErrResp{Message: err.Error()})
			return
		}
		slog.Error("login", "error", err)
		c.JSON(http.StatusInternalServerError, errresp.HttpErrResp{Message: errresp.ErrInternal.Error()})
		return
	}

	c.JSON(http.StatusOK, tokenPairResp{AccessToken: pair.AccessToken, RefreshToken: pair.RefreshToken})
}
