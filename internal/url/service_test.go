package url

import (
	"context"
	"errors"
	"testing"
	"time"
)

type repositoryStub struct {
	created ShortURL
	err     error
}

func (r *repositoryStub) Create(_ context.Context, code, originalURL, _ string) (ShortURL, error) {
	if r.err != nil {
		return ShortURL{}, r.err
	}
	r.created = ShortURL{Code: code, OriginalURL: originalURL, CreatedAt: time.Now()}
	return r.created, nil
}

func (r *repositoryStub) FindByCode(context.Context, string) (ShortURL, error) {
	return r.created, nil
}

func (r *repositoryStub) List(context.Context, string, bool, int, int, string, string) ([]ShortURL, int, error) {
	return []ShortURL{r.created}, 1, r.err
}

func (r *repositoryStub) FindByID(context.Context, string, string, bool) (ShortURL, error) {
	return r.created, r.err
}

func (r *repositoryStub) Disable(context.Context, string, string, bool) error {
	return r.err
}

func (r *repositoryStub) Enable(context.Context, string, string, bool) error {
	return r.err
}

func (r *repositoryStub) RecordClick(context.Context, string, string, string) error {
	return r.err
}

func TestServiceCreateTrimsAndStoresValidURL(t *testing.T) {
	repository := &repositoryStub{}
	service := NewService(repository)

	item, err := service.Create(context.Background(), "  https://example.com/path  ", "")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if item.OriginalURL != "https://example.com/path" {
		t.Fatalf("OriginalURL = %q, want trimmed URL", item.OriginalURL)
	}
	if len(item.Code) != 7 {
		t.Fatalf("code length = %d, want 7", len(item.Code))
	}
}

func TestServiceCreateRejectsInvalidURL(t *testing.T) {
	service := NewService(&repositoryStub{})

	_, err := service.Create(context.Background(), "ftp://example.com", "")
	if !errors.Is(err, ErrInvalidURL) {
		t.Fatalf("Create() error = %v, want ErrInvalidURL", err)
	}
}

func TestIsReservedCode(t *testing.T) {
	for _, code := range []string{"login", "dashboard", "api", "healthz", "static", "audit"} {
		if !isReserved(code) {
			t.Errorf("isReserved(%q) = false, want true", code)
		}
	}
	if isReserved("abc1234") {
		t.Fatal("isReserved(abc1234) = true, want false")
	}
}
