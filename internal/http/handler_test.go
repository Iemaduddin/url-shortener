package http

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	urlservice "url-shortener/internal/url"
)

type repositoryStub struct {
	item        urlservice.ShortURL
	err         error
	recordedIDs []string
}

func (r *repositoryStub) Create(_ context.Context, code, originalURL, _ string) (urlservice.ShortURL, error) {
	r.item = urlservice.ShortURL{Code: code, OriginalURL: originalURL, CreatedAt: time.Now()}
	return r.item, r.err
}

func (r *repositoryStub) FindByCode(_ context.Context, code string) (urlservice.ShortURL, error) {
	if r.err != nil {
		return urlservice.ShortURL{}, r.err
	}
	if r.item.Code != code {
		return urlservice.ShortURL{}, urlservice.ErrNotFound
	}
	return r.item, nil
}

func (r *repositoryStub) List(context.Context, string, bool, int, int, string, string) ([]urlservice.ShortURL, int, error) {
	if r.err != nil {
		return nil, 0, r.err
	}

	if r.item.ID == "" {
		return []urlservice.ShortURL{}, 0, nil
	}
	return []urlservice.ShortURL{r.item}, 1, nil
}

func (r *repositoryStub) FindByID(context.Context, string, string, bool) (urlservice.ShortURL, error) {
	if r.err != nil {
		return urlservice.ShortURL{}, r.err
	}
	return r.item, nil
}

func (r *repositoryStub) Disable(context.Context, string, string, bool) error {
	return r.err
}

func (r *repositoryStub) Enable(context.Context, string, string, bool) error {
	return r.err
}

func (r *repositoryStub) RecordClick(_ context.Context, id, _, _ string) error {
	r.recordedIDs = append(r.recordedIDs, id)
	return r.err
}

func newTestHandler(repository *repositoryStub) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewHandler(urlservice.NewService(repository), nil, "http://localhost:8080", logger)
}

func TestCreateEndpointReturnsShortURL(t *testing.T) {
	handler := newTestHandler(&repositoryStub{})
	request := httptest.NewRequest(http.MethodPost, "/api/urls", strings.NewReader(`{"original_url":"https://example.com"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), `"short_url":"http://localhost:8080/`) {
		t.Fatalf("response = %q, want short URL", response.Body.String())
	}
}

func TestCreateEndpointRejectsInvalidURL(t *testing.T) {
	handler := newTestHandler(&repositoryStub{})
	request := httptest.NewRequest(http.MethodPost, "/api/urls", strings.NewReader(`{"original_url":"ftp://example.com"}`))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if !strings.Contains(response.Body.String(), `"code":"INVALID_URL"`) {
		t.Fatalf("response = %q, want INVALID_URL", response.Body.String())
	}
}

func TestRedirectEndpointRedirectsToOriginalURL(t *testing.T) {
	repository := &repositoryStub{
		item: urlservice.ShortURL{Code: "abc1234", OriginalURL: "https://example.com"},
	}
	handler := newTestHandler(repository)
	request := httptest.NewRequest(http.MethodGet, "/abc1234", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusFound)
	}
	if location := response.Header().Get("Location"); location != "https://example.com" {
		t.Fatalf("Location = %q, want original URL", location)
	}
	if len(repository.recordedIDs) != 1 || repository.recordedIDs[0] != repository.item.ID {
		t.Fatalf("recorded click IDs = %v, want %q", repository.recordedIDs, repository.item.ID)
	}
}

func TestRedirectEndpointReturnsNotFound(t *testing.T) {
	repository := &repositoryStub{err: urlservice.ErrNotFound}
	handler := newTestHandler(repository)
	request := httptest.NewRequest(http.MethodGet, "/missing", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestRateLimitKeepsQRPreviewBucketSeparate(t *testing.T) {
	handler := newTestHandler(&repositoryStub{})

	for i := 0; i < 300; i++ {
		request := httptest.NewRequest(http.MethodPut, "/api/urls/11111111-1111-1111-1111-111111111111/qr/preview", nil)
		response := httptest.NewRecorder()

		handler.ServeHTTP(response, request)

		if response.Code == http.StatusTooManyRequests {
			t.Fatalf("preview request %d was rate limited before the preview limit of 300", i+1)
		}
	}

	request := httptest.NewRequest(http.MethodGet, "/api/urls", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code == http.StatusTooManyRequests {
		t.Fatal("preview requests consumed the general rate limit bucket")
	}
}
