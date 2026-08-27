package http

import (
	"errors"
	"net/http"

	httpdto "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/entity/http"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/errorsx"
	errresp "github.com/aleksiaichuk-innowise/inno_taxi/shared/errorsx"
	"github.com/gin-gonic/gin"
)

func (h Handler) Profile(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, errresp.HttpErrResp{
			Message: "invalid user_id",
		})
		return
	}
	p, err := h.svc.GetProfileByUser(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, errorsx.ErrDriverNotFound) {
			c.JSON(http.StatusNotFound, errresp.HttpErrResp{Message: err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, errresp.HttpErrResp{Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, httpdto.DriverResp{
		ID:       p.ID,
		UserID:   p.UserID,
		TaxiType: string(p.TaxiType),
		Status:   string(p.Status),
	})
}
