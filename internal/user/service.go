package user

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidEmail       = errors.New("invalid email")
	ErrInvalidRole        = errors.New("invalid role")
)

const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

type Service struct{ repository Repository }

func NewService(repository Repository) *Service { return &Service{repository: repository} }

func (s *Service) Register(ctx context.Context, email, password string) (User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if !strings.Contains(email, "@") || len(password) < 8 {
		return User{}, ErrInvalidEmail
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, err
	}
	return s.repository.CreateUser(ctx, email, string(hash))
}

func (s *Service) EnsureDefaultUser(ctx context.Context, email, password, role string) (bool, error) {
	if role != RoleUser && role != RoleAdmin {
		return false, errors.New("invalid role")
	}
	email = strings.ToLower(strings.TrimSpace(email))
	if !strings.Contains(email, "@") || len(password) < 8 {
		return false, ErrInvalidEmail
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return false, err
	}
	return s.repository.EnsureUser(ctx, email, string(hash), role)
}

func (s *Service) ListUsers(ctx context.Context) ([]User, error) {
	return s.repository.ListUsers(ctx)
}

func (s *Service) CreateManagedUser(ctx context.Context, email, password, role string) (User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if !strings.Contains(email, "@") || len(password) < 8 {
		return User{}, ErrInvalidEmail
	}
	if role != RoleUser && role != RoleAdmin {
		return User{}, ErrInvalidRole
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, err
	}
	return s.repository.CreateManagedUser(ctx, email, string(hash), role)
}

func (s *Service) UpdateUser(ctx context.Context, id, email, password, role string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if !strings.Contains(email, "@") || (password != "" && len(password) < 8) {
		return ErrInvalidEmail
	}
	if role != RoleUser && role != RoleAdmin {
		return ErrInvalidRole
	}
	hash := ""
	if password != "" {
		var err error
		hashBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		hash = string(hashBytes)
	}
	return s.repository.UpdateUser(ctx, id, email, hash, role)
}

func (s *Service) UpdateStatus(ctx context.Context, id, status string) error {
	if status != "active" && status != "disabled" {
		return errors.New("invalid status")
	}
	return s.repository.UpdateStatus(ctx, id, status)
}

func (s *Service) DeleteUser(ctx context.Context, id string) error {
	return s.repository.DeleteUser(ctx, id)
}

func (s *Service) Login(ctx context.Context, email, password string) (Session, error) {
	user, passwordHash, err := s.repository.FindByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil || user.Status != "active" || bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)) != nil {
		return Session{}, ErrInvalidCredentials
	}
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return Session{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)
	session := Session{Token: token, UserID: user.ID, ExpiresAt: time.Now().Add(24 * time.Hour)}
	if err := s.repository.CreateSession(ctx, token, user.ID, session.ExpiresAt); err != nil {
		return Session{}, err
	}
	return session, nil
}

func (s *Service) Session(ctx context.Context, token string) (Session, error) {
	return s.repository.FindSession(ctx, token)
}

func (s *Service) Logout(ctx context.Context, token string) error {
	return s.repository.DeleteSession(ctx, token)
}
