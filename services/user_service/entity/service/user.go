package service

import "time"

type Role string

const (
	RoleUser    Role = "user"
	RoleDriver  Role = "driver"
	RoleAdmin   Role = "admin"
	RoleAnalyst Role = "analyst"
)

func (r Role) IsValid() bool {
	switch r {
	case RoleUser, RoleDriver, RoleAdmin, RoleAnalyst:
		return true
	default:
		return false
	}
}

func RolesToStrings(roles []Role) []string {
	out := make([]string, len(roles))
	for i, r := range roles {
		out[i] = string(r)
	}
	return out
}

type User struct {
	ID           string
	Name         string
	Email        string
	Phone        string
	PasswordHash string
	Roles        []Role
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}

type RegisterInput struct {
	Name     string
	Email    string
	Phone    string
	Password string
	Role     Role
}

type ProfileInput struct {
	Name  string
	Email string
	Phone string
}
