package service

import "time"

type Role string

const (
	RoleUser   Role = "user"
	RoleDriver Role = "driver"
)

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
	Name            string
	Email           string
	Phone           string
	CurrentPassword string
}
