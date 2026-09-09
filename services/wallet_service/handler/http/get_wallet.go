package http

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/errorsx"
	errresp "github.com/aleksiaichuk-innowise/inno_taxi/shared/errorsx"
	"github.com/gin-gonic/gin"
)

func (h *Handler) GetWallet(c *gin.Context) {
	userID := c.Param("user_id")

	wallet, err := h.svc.GetWallet(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, errorsx.ErrWalletNotFound) {
			c.JSON(http.StatusNotFound, errresp.HttpErrResp{Message: err.Error()})
			return
		}
		slog.Error("get wallet", "error", err)
		c.JSON(http.StatusInternalServerError, errresp.HttpErrResp{Message: errresp.ErrInternal.Error()})
		return
	}

	c.JSON(http.StatusOK, walletRespFrom(wallet))
}
