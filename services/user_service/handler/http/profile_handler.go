package http

import (
	"errors"
	"log/slog"
	"net/http"

	resp "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/entity/http"
	"github.com/aleksiaichuk-innowise/inno_taxi/shared/errorsx"
	"github.com/gin-gonic/gin"
)

func (h *Handler) Profile(c *gin.Context) {
	id := c.Param("id")
	usr, err := h.userSvc.GetProfile(c.Request.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, errorsx.ErrUserNotFound):
			c.JSON(http.StatusNotFound, errorsx.HttpErrResp{Message: err.Error()})
		default:
			slog.Error("get profile", "error", err)
			c.JSON(http.StatusInternalServerError, errorsx.HttpErrResp{Message: errorsx.ErrInternal.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, resp.UserResp{
		ID:    usr.ID,
		Name:  usr.Name,
		Email: usr.Email,
		Phone: usr.Phone,
		Roles: rolesToStrings(usr.Roles),
	})
}
