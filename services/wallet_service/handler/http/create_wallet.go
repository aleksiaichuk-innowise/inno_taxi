package http

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/errorsx"
	errresp "github.com/aleksiaichuk-innowise/inno_taxi/shared/errorsx"
	"github.com/gin-gonic/gin"
)

type createWalletReq struct {
	UserID string `json:"user_id" binding:"required"`
}

func (h *Handler) CreateWallet(c *gin.Context) {
	var req createWalletReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errresp.HttpErrResp{Message: err.Error()})
		return
	}

	wallet, err := h.svc.CreateWallet(c.Request.Context(), req.UserID)
	if err != nil {
		if errors.Is(err, errorsx.ErrWalletAlreadyExists) {
			c.JSON(http.StatusUnprocessableEntity, errresp.HttpErrResp{Message: err.Error()})
			return
		}
		slog.Error("create wallet", "error", err)
		c.JSON(http.StatusInternalServerError, errresp.HttpErrResp{Message: errresp.ErrInternal.Error()})
		return
	}

	c.JSON(http.StatusCreated, walletRespFrom(wallet))
}
