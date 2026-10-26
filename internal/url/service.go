package url

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

const alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

var ErrInvalidURL = errors.New("invalid URL")

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Create(ctx context.Context, originalURL, userID string) (ShortURL, error) {
	originalURL = strings.TrimSpace(originalURL)
	if err := validateURL(originalURL); err != nil {
		return ShortURL{}, err
	}

	for attempt := 0; attempt < 3; attempt++ {
		code, err := newCode(7)
		if err != nil {
			return ShortURL{}, fmt.Errorf("generate code: %w", err)
		}
		item, err := s.repository.Create(ctx, code, originalURL, userID)
		if err == nil {
			return item, nil
		}
		if !isUniqueViolation(err) {
			return ShortURL{}, err
		}
	}
	return ShortURL{}, errors.New("could not generate unique code")
}

func (s *Service) FindByCode(ctx context.Context, code string) (ShortURL, error) {
	return s.repository.FindByCode(ctx, code)
}

func (s *Service) List(ctx context.Context, userID string, admin bool, page, pageSize int, search, status string) ([]ShortURL, int, error) {
	return s.repository.List(ctx, userID, admin, page, pageSize, search, status)
}

func (s *Service) FindByID(ctx context.Context, id, userID string, admin bool) (ShortURL, error) {
	return s.repository.FindByID(ctx, id, userID, admin)
}

func (s *Service) FindPublicByID(ctx context.Context, id string) (ShortURL, error) {
	return s.repository.FindByID(ctx, id, "", true)
}

func (s *Service) Disable(ctx context.Context, id, userID string, admin bool) error {
	return s.repository.Disable(ctx, id, userID, admin)
}

func (s *Service) Enable(ctx context.Context, id, userID string, admin bool) error {
	return s.repository.Enable(ctx, id, userID, admin)
}

func (s *Service) RecordClick(ctx context.Context, id, userAgent, referer string) error {
	return s.repository.RecordClick(ctx, id, userAgent, referer)
}

func validateURL(value string) error {
	if len(value) == 0 || len(value) > 2048 {
		return ErrInvalidURL
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return ErrInvalidURL
	}
	return nil
}

func newCode(length int) (string, error) {
	result := make([]byte, length)
	random := make([]byte, length)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	for i, value := range random {
		result[i] = alphabet[int(value)%len(alphabet)]
	}
	return string(result), nil
}

func isUniqueViolation(err error) bool {
	var pgError *pgconn.PgError
	return errors.As(err, &pgError) && pgError.Code == "23505" && pgError.ConstraintName == "short_urls_code_key"
}
