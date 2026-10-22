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

func (r *repositoryStub) List(context.Context, string, bool) ([]ShortURL, error) {
	return []ShortURL{r.created}, r.err
}

func (r *repositoryStub) Disable(context.Context, string, string, bool) error {
	return r.err
}

func (r *repositoryStub) Enable(context.Context, string, string, bool) error {
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
