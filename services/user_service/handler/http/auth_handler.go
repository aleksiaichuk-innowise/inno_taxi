package http

import (
	"errors"
	"log/slog"
	"net/http"

	httpEntity "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/entity/http"
	serviceEntity "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/shared/errorsx"

	"github.com/gin-gonic/gin"
)

func (h *Handler) Register(c *gin.Context) {
	var req httpEntity.RegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorsx.HttpErrResp{Message: err.Error()})
		return
	}
	if err := h.validate.Struct(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, errorsx.HttpErrResp{Message: err.Error()})
		return
	}

	input := serviceEntity.RegisterInput{
		Name:     req.Name,
		Email:    req.Email,
		Phone:    req.Phone,
		Password: req.Password,
		Role:     serviceEntity.Role(req.Role),
	}

	user, err := h.userSvc.CreateUser(c.Request.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, errorsx.ErrUserAlreadyExists):
			c.JSON(http.StatusConflict, errorsx.HttpErrResp{Message: err.Error()})
		case errors.Is(err, errorsx.ErrInvalidRole):
			c.JSON(http.StatusUnprocessableEntity, errorsx.HttpErrResp{Message: err.Error()})
		default:
			slog.Error("register: create user", "error", err)
			c.JSON(http.StatusInternalServerError, errorsx.HttpErrResp{Message: errorsx.ErrInternal.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, httpEntity.UserResp{
		ID:    user.ID,
		Name:  user.Name,
		Email: user.Email,
		Phone: user.Phone,
		Roles: rolesToStrings(user.Roles),
	})
}

func rolesToStrings(roles []serviceEntity.Role) []string {
	out := make([]string, len(roles))
	for i, r := range roles {
		out[i] = string(r)
	}
	return out
}

func (h *Handler) VerifyCredentials(c *gin.Context) {
	var req httpEntity.VerifyCredentialsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorsx.HttpErrResp{Message: err.Error()})
		return
	}
	if err := h.validate.Struct(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, errorsx.HttpErrResp{Message: err.Error()})
		return
	}

	user, err := h.userSvc.VerifyCredentials(c.Request.Context(), req.Login, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, errorsx.ErrInvalidCredentials):
			c.JSON(http.StatusUnauthorized, errorsx.HttpErrResp{Message: err.Error()})
		default:
			slog.Error("verify credentials", "error", err)
			c.JSON(http.StatusInternalServerError, errorsx.HttpErrResp{Message: errorsx.ErrInternal.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, httpEntity.UserResp{
		ID:    user.ID,
		Name:  user.Name,
		Email: user.Email,
		Phone: user.Phone,
		Roles: rolesToStrings(user.Roles),
	})
}
