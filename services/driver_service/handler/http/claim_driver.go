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

type claimDriverReq struct {
	TaxiType string `json:"taxi_type" binding:"required"`
}

func (h Handler) ClaimDriver(c *gin.Context) {
	var req claimDriverReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errresp.HttpErrResp{Message: err.Error()})
		return
	}

	d, err := h.svc.ClaimAvailableDriver(c.Request.Context(), service.TaxiType(req.TaxiType))
	if err != nil {
		switch {
		case errors.Is(err, errorsx.ErrNoAvailableDriver):
			c.JSON(http.StatusNotFound, errresp.HttpErrResp{Message: err.Error()})
		case errors.Is(err, errorsx.ErrInvalidTaxiType):
			c.JSON(http.StatusUnprocessableEntity, errresp.HttpErrResp{Message: err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, errresp.HttpErrResp{Message: err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, http_dto.DriverResp{
		ID:       d.ID,
		UserID:   d.UserID,
		TaxiType: string(d.TaxiType),
		Status:   string(d.Status),
	})
}
