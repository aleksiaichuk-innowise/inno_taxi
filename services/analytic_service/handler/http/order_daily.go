package http

import (
	"errors"
	"log/slog"

	"github.com/aleksiaichuk-innowise/inno_taxi/services/analytic_service/errorsx"
	errresp "github.com/aleksiaichuk-innowise/inno_taxi/shared/errorsx"
	"github.com/gofiber/fiber/v2"
)

type dailyOrderCountResp struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

func (h *Handler) GetDailyOrderCounts(c *fiber.Ctx) error {
	from, to, err := parseDateRange(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errresp.HttpErrResp{Message: err.Error()})
	}

	counts, err := h.svc.GetDailyOrderCounts(c.Context(), from, to)
	if err != nil {
		if errors.Is(err, errorsx.ErrInvalidDateRange) {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(errresp.HttpErrResp{Message: err.Error()})
		}
		slog.Error("get daily order counts", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(errresp.HttpErrResp{Message: errresp.ErrInternal.Error()})
	}

	resp := make([]dailyOrderCountResp, 0, len(counts))
	for _, d := range counts {
		resp = append(resp, dailyOrderCountResp{Date: d.Date, Count: d.Count})
	}
	return c.JSON(resp)
}
