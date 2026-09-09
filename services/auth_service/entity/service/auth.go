package service

import "github.com/golang-jwt/jwt/v5"

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

type AccessClaims struct {
	UserID string   `json:"user_id"`
	Roles  []string `json:"roles"`
	jwt.RegisteredClaims
}

type RefreshClaims struct {
	UserID string   `json:"user_id"`
	Roles  []string `json:"roles"`
	jwt.RegisteredClaims
}
