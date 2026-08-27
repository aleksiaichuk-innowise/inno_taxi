package http

import (
	"errors"
	"net/http"

	http_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/entity/http"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/errorsx"
	errresp "github.com/aleksiaichuk-innowise/inno_taxi/shared/errorsx"
	"github.com/gin-gonic/gin"
)

func (h Handler) GetDrivers(c *gin.Context) {
	statusQuery := c.Query("status")
	drivers, err := h.svc.GetDriversByStatus(c.Request.Context(), service.Status(statusQuery))
	if err != nil {
		if errors.Is(err, errorsx.ErrInvalidStatus) {
			c.JSON(http.StatusBadRequest, errresp.HttpErrResp{
				Message: err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, errresp.HttpErrResp{
			Message: "internal server error",
		})
		return
	}
	resp := make([]http_dto.DriverResp, 0, len(drivers))
	for _, d := range drivers {
		resp = append(resp, http_dto.DriverResp{
			ID:       d.ID,
			UserID:   d.UserID,
			TaxiType: string(d.TaxiType),
			Status:   string(d.Status),
		})
	}
	c.JSON(http.StatusOK, resp)
}
