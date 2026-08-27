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

func (h Handler) UpdateType(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, errresp.HttpErrResp{
			Message: "invalid user_id",
		})
		return
	}
	var req http_dto.UpdateDriverTypeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errresp.HttpErrResp{
			Message: err.Error(),
		})
		return
	}
	if err := h.validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, errresp.HttpErrResp{
			Message: err.Error(),
		})
		return
	}
	err := h.svc.UpdateTaxiTypeByUser(c.Request.Context(), userID, service.TaxiType(req.Type))
	if err != nil {
		if errors.Is(err, errorsx.ErrDriverNotFound) {
			c.JSON(http.StatusNotFound, errresp.HttpErrResp{
				Message: err.Error(),
			})
			return
		}
		if errors.Is(err, errorsx.ErrInvalidTaxiType) {
			c.JSON(http.StatusUnprocessableEntity, errresp.HttpErrResp{
				Message: err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, errresp.HttpErrResp{
			Message: err.Error(),
		})
		return
	}
	c.Status(http.StatusOK)

}
