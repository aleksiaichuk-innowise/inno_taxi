package http

import (
	"log/slog"

	errresp "github.com/aleksiaichuk-innowise/inno_taxi/shared/errorsx"
	"github.com/gofiber/fiber/v2"
)

type driverRatingResp struct {
	Average float64 `json:"average"`
	Count   int64   `json:"count"`
}

func (h *Handler) GetDriverRatingStats(c *fiber.Ctx) error {
	driverID := c.Params("driver_id")

	stats, err := h.svc.GetDriverRatingStats(c.Context(), driverID)
	if err != nil {
		slog.Error("get driver rating stats", "error", err, "driver_id", driverID)
		return c.Status(fiber.StatusInternalServerError).JSON(errresp.HttpErrResp{Message: errresp.ErrInternal.Error()})
	}

	return c.JSON(driverRatingResp{Average: stats.Average, Count: stats.Count})
}
