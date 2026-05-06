package domain

import "time"

type User struct {
	ID           string
	Email        string
	PasswordHash string
	Name         string
	PhoneNumber  string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
