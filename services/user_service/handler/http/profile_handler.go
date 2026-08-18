package http

import (
	"errors"
	"log/slog"
	"net/http"

	resp "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/entity/http"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/entity/service"
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

func (h *Handler) UpdateProfile(c *gin.Context) {
	id := c.Param("id")
	var req resp.UpdateProfileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorsx.HttpErrResp{Message: err.Error()})
		return
	}
	if err := h.validate.Struct(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, errorsx.HttpErrResp{Message: err.Error()})
		return
	}

	input := service.ProfileInput{
		Name:            req.Name,
		Email:           req.Email,
		Phone:           req.Phone,
		CurrentPassword: req.CurrentPassword,
	}

	usr, err := h.userSvc.UpdateProfile(c.Request.Context(), id, input)
	if err != nil {
		switch {
		case errors.Is(err, errorsx.ErrUserNotFound):
			c.JSON(http.StatusNotFound, errorsx.HttpErrResp{Message: err.Error()})
		case errors.Is(err, errorsx.ErrInvalidCredentials):
			c.JSON(http.StatusUnauthorized, errorsx.HttpErrResp{Message: err.Error()})
		case errors.Is(err, errorsx.ErrUserAlreadyExists):
			c.JSON(http.StatusConflict, errorsx.HttpErrResp{Message: err.Error()})
		default:
			slog.Error("update profile", "error", err)
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

func (h *Handler) UpdatePassword(c *gin.Context) {
	id := c.Param("id")
	var req resp.UpdatePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorsx.HttpErrResp{Message: err.Error()})
		return
	}
	if err := h.validate.Struct(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, errorsx.HttpErrResp{Message: err.Error()})
		return
	}

	err := h.userSvc.UpdatePassword(c.Request.Context(), id, req.CurrentPassword, req.NewPassword)

	if err != nil {
		switch {
		case errors.Is(err, errorsx.ErrUserNotFound):
			c.JSON(http.StatusNotFound, errorsx.HttpErrResp{Message: err.Error()})
		case errors.Is(err, errorsx.ErrInvalidCredentials):
			c.JSON(http.StatusUnauthorized, errorsx.HttpErrResp{Message: err.Error()})
		default:
			slog.Error("update profile", "error", err)
			c.JSON(http.StatusInternalServerError, errorsx.HttpErrResp{Message: errorsx.ErrInternal.Error()})
		}
		return
	}

}
