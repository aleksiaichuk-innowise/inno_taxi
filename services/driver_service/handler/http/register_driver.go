package http

import (
	"errors"
	"log/slog"
	"net/http"

	http_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/entity/http"
	input "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/errorsx"
	errresp "github.com/aleksiaichuk-innowise/inno_taxi/shared/errorsx"
	"github.com/gin-gonic/gin"
)

func (h Handler) Register(c *gin.Context) {
	var req http_dto.CreateDriverReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errresp.HttpErrResp{Message: err.Error()})
		return
	}

	if err := h.validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, errresp.HttpErrResp{Message: err.Error()})
		return
	}

	dto := &input.CreateDriverInput{
		UserID:   req.UserID,
		TaxiType: input.TaxiType(req.TaxiType),
	}

	d, err := h.svc.CreateDriver(c.Request.Context(), dto)
	if err != nil {
		switch {
		case errors.Is(err, errorsx.ErrDriverAlreadyExists):
			c.JSON(http.StatusUnprocessableEntity, errresp.HttpErrResp{Message: "driver already exists"})
		case errors.Is(err, errorsx.ErrInvalidTaxiType):
			c.JSON(http.StatusUnprocessableEntity, errresp.HttpErrResp{Message: err.Error()})
		default:
			slog.Error("Failed to register driver", "error", err)
			c.JSON(http.StatusInternalServerError, errresp.HttpErrResp{Message: err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, http_dto.DriverResp{
		ID:       d.ID,
		UserID:   d.UserID,
		TaxiType: string(d.TaxiType),
		Status:   string(d.Status),
	})

}
