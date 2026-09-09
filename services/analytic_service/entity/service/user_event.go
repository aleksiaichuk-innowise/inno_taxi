package service

import "time"

type UserRegisteredEvent struct {
	UserID       string
	Name         string
	Email        string
	Phone        string
	Role         string
	RegisteredAt time.Time
}
