package user

import "time"

type User struct {
	ID        string
	Email     string
	Role      string
	Status    string
	CreatedAt time.Time
}

type Session struct {
	Token     string
	UserID    string
	Role      string
	ExpiresAt time.Time
}
