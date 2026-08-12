package service

import "time"

type Role string

type User struct {
	ID           string
	Name         string
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
