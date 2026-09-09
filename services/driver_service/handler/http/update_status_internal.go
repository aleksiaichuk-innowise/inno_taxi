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

// UpdateStatusInternal is the network-trusted counterpart to UpdateStatus:
// the user_id comes from the URL, not a JWT, because callers here (e.g.
// order_service releasing a driver on cancellation) act on a driver's
// behalf without one.
func (h Handler) UpdateStatusInternal(c *gin.Context) {
	userID := c.Param("user_id")

	var req http_dto.UpdateDriverStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errresp.HttpErrResp{Message: err.Error()})
		return
	}
	if err := h.validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, errresp.HttpErrResp{Message: err.Error()})
		return
	}

	err := h.svc.UpdateStatusByUser(c.Request.Context(), userID, service.Status(req.Status))
	if err != nil {
		switch {
		case errors.Is(err, errorsx.ErrDriverNotFound):
			c.JSON(http.StatusNotFound, errresp.HttpErrResp{Message: err.Error()})
		case errors.Is(err, errorsx.ErrInvalidStatus):
			c.JSON(http.StatusUnprocessableEntity, errresp.HttpErrResp{Message: err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, errresp.HttpErrResp{Message: err.Error()})
		}
		return
	}
	c.Status(http.StatusOK)
}
