package http

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/errorsx"
	errresp "github.com/aleksiaichuk-innowise/inno_taxi/shared/errorsx"
	"github.com/gin-gonic/gin"
)

func (h *Handler) ListTransactions(c *gin.Context) {
	userID := c.Param("user_id")

	txs, err := h.svc.ListRecentTransactions(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, errorsx.ErrWalletNotFound) {
			c.JSON(http.StatusNotFound, errresp.HttpErrResp{Message: err.Error()})
			return
		}
		slog.Error("list transactions", "error", err)
		c.JSON(http.StatusInternalServerError, errresp.HttpErrResp{Message: errresp.ErrInternal.Error()})
		return
	}

	resp := make([]transactionResp, 0, len(txs))
	for _, t := range txs {
		resp = append(resp, transactionRespFrom(t))
	}
	c.JSON(http.StatusOK, resp)
}
