package http

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/errorsx"
	errresp "github.com/aleksiaichuk-innowise/inno_taxi/shared/errorsx"
	"github.com/gin-gonic/gin"
)

type chargeReq struct {
	AmountMinorUnits int64  `json:"amount_minor_units" binding:"required"`
	ReferenceID      string `json:"reference_id" binding:"required"`
}

func (h *Handler) Charge(c *gin.Context) {
	userID := c.Param("user_id")

	var req chargeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errresp.HttpErrResp{Message: err.Error()})
		return
	}

	tx, created, err := h.svc.Charge(c.Request.Context(), userID, req.AmountMinorUnits, req.ReferenceID)
	if err != nil {
		switch {
		case errors.Is(err, errorsx.ErrWalletNotFound):
			c.JSON(http.StatusNotFound, errresp.HttpErrResp{Message: err.Error()})
		case errors.Is(err, errorsx.ErrInvalidAmount):
			c.JSON(http.StatusUnprocessableEntity, errresp.HttpErrResp{Message: err.Error()})
		case errors.Is(err, errorsx.ErrInsufficientFunds):
			c.JSON(http.StatusPaymentRequired, errresp.HttpErrResp{Message: err.Error()})
		default:
			slog.Error("charge wallet", "error", err)
			c.JSON(http.StatusInternalServerError, errresp.HttpErrResp{Message: errresp.ErrInternal.Error()})
		}
		return
	}

	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	c.JSON(status, transactionRespFrom(tx))
}
