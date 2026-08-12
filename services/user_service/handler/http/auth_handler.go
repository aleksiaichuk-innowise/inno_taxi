package http

import (
	http2 "net/http"

	"github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/entity/http"
	"github.com/aleksiaichuk-innowise/inno_taxi/shared/errorsx"
	"github.com/gin-gonic/gin"
)

func (h *Handler) Register(c *gin.Context) {
	var r http.RegisterReq
	if err := c.ShouldBindJSON(&r); err != nil {
		c.JSON(http2.StatusUnprocessableEntity, errorsx.HttpErrResp{message: err.Error()})
		return
	}

}

func (h *Handler) Login(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "user login",
	})
}
