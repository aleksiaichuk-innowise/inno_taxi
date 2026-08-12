package http

import (
	"log/slog"
	http2 "net/http"

	"github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/entity/http"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/shared/errorsx"

	"github.com/gin-gonic/gin"
)

func (h *Handler) Register(c *gin.Context) {
	var r http.RegisterReq
	if err := c.ShouldBindJSON(&r); err != nil {
		c.JSON(http2.StatusUnprocessableEntity, errorsx.HttpErrResp{Message: "Unprocessable Entity"})
		return
	}
	if err := h.validate.Struct(&r); err != nil {
		c.JSON(http2.StatusUnprocessableEntity, errorsx.HttpErrResp{Message: err.Error()})
	}


	input := service.RegisterInput{
		Name: r.Name,
		Phone: r.Phone,
		Password: r.Password,
		Email: r.Email,
		Role:  service.Role(r.Role),
	}



	if err := h.userSvc.CreateUser(c, input); err != nil {
		slog.Info("register error", err)
	}

}

func (h *Handler) Login(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "user login",
	})
}
