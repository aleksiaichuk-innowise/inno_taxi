package user_service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	gateway_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/auth_service/entity/gateway"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/auth_service/errorsx"
)

type UserServiceGateway interface {
	VerifyCredentials(ctx context.Context, login, password string) (gateway_dto.UserInfo, error)
}

type gateway struct {
	baseURL string
	client  *http.Client
}

func NewUserServiceGateway(baseURL string, client *http.Client) UserServiceGateway {
	if client == nil {
		client = http.DefaultClient
	}
	return &gateway{baseURL: baseURL, client: client}
}

type verifyCredentialsReq struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type userResp struct {
	ID    string   `json:"id"`
	Roles []string `json:"roles"`
}

func (g *gateway) VerifyCredentials(ctx context.Context, login, password string) (gateway_dto.UserInfo, error) {
	body, err := json.Marshal(verifyCredentialsReq{Login: login, Password: password})
	if err != nil {
		return gateway_dto.UserInfo{}, fmt.Errorf("marshal verify credentials request: %w", err)
	}

	url := g.baseURL + "/internal/verify-credentials"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return gateway_dto.UserInfo{}, fmt.Errorf("build verify credentials request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return gateway_dto.UserInfo{}, fmt.Errorf("call user service: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var u userResp
		if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
			return gateway_dto.UserInfo{}, fmt.Errorf("decode verify credentials response: %w", err)
		}
		return gateway_dto.UserInfo{ID: u.ID, Roles: u.Roles}, nil
	case http.StatusUnauthorized:
		return gateway_dto.UserInfo{}, errorsx.ErrInvalidCredentials
	default:
		return gateway_dto.UserInfo{}, fmt.Errorf("user service returned unexpected status %d", resp.StatusCode)
	}
}
