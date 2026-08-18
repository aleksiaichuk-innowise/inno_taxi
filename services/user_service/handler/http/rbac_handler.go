package http

import (
	"log/slog"
	"net/http"

	"github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/shared/errorsx"
	"github.com/gin-gonic/gin"
)

func (h *Handler) AssignRoleAnalytic(c *gin.Context) {
	id := c.Param("id")
	if err := h.userSvc.AssignRole(c.Request.Context(), id, service.RoleAnalyst); err != nil {
		slog.Error("assign role analytic", "error", err)
		c.JSON(http.StatusInternalServerError, errorsx.HttpErrResp{Message: errorsx.ErrInternal.Error()})
		return
	}
	c.JSON(http.StatusOK, errorsx.HttpErrResp{Message: "success"})
}
