package http

import (
	"errors"
	"log/slog"

	"github.com/aleksiaichuk-innowise/inno_taxi/services/analytic_service/errorsx"
	errresp "github.com/aleksiaichuk-innowise/inno_taxi/shared/errorsx"
	"github.com/gofiber/fiber/v2"
)

type orderStatsResp struct {
	TotalOrders      int64            `json:"total_orders"`
	CountsByStatus   map[string]int64 `json:"counts_by_status"`
	CountsByTaxiType map[string]int64 `json:"counts_by_taxi_type"`
}

func (h *Handler) GetOrderStats(c *fiber.Ctx) error {
	from, to, err := parseDateRange(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errresp.HttpErrResp{Message: err.Error()})
	}

	stats, err := h.svc.GetOrderStats(c.Context(), from, to)
	if err != nil {
		if errors.Is(err, errorsx.ErrInvalidDateRange) {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(errresp.HttpErrResp{Message: err.Error()})
		}
		slog.Error("get order stats", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(errresp.HttpErrResp{Message: errresp.ErrInternal.Error()})
	}

	return c.JSON(orderStatsResp{
		TotalOrders:      stats.TotalOrders,
		CountsByStatus:   stats.CountsByStatus,
		CountsByTaxiType: stats.CountsByTaxiType,
	})
}
