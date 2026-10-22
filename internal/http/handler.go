package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	urlservice "url-shortener/internal/url"
)

type Handler struct {
	service       *urlservice.Service
	publicBaseURL string
	logger        *slog.Logger
}

type createURLRequest struct {
	OriginalURL string `json:"original_url"`
}

type createURLResponse struct {
	ShortURL string `json:"short_url"`
}

type errorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func NewHandler(service *urlservice.Service, publicBaseURL string, logger *slog.Logger) http.Handler {
	h := &Handler{
		service:       service,
		publicBaseURL: strings.TrimRight(publicBaseURL, "/"),
		logger:        logger,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", h.route)
	mux.HandleFunc("/healthz", h.health)
	mux.HandleFunc("/api/urls", h.create)
	return logging(mux, logger)
}

func (h *Handler) route(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/assets/") {
		http.StripPrefix("/", http.FileServer(http.Dir("web/dist"))).ServeHTTP(w, r)
		return
	}
	if r.URL.Path == "/" {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		http.ServeFile(w, r, "web/dist/index.html")
		return
	}
	if r.Method == http.MethodGet {
		h.redirect(w, r)
		return
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var request createURLRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Request tidak valid.")
		return
	}
	item, err := h.service.Create(r.Context(), request.OriginalURL)
	if err != nil {
		if errors.Is(err, urlservice.ErrInvalidURL) {
			h.writeError(w, http.StatusBadRequest, "INVALID_URL", "Masukkan URL HTTP atau HTTPS yang valid.")
			return
		}
		h.logger.Error("create short URL failed", "error", err)
		h.writeError(w, http.StatusInternalServerError, "CREATE_FAILED", "Short URL gagal dibuat.")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(createURLResponse{ShortURL: h.publicBaseURL + "/" + item.Code})
}

func (h *Handler) redirect(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimPrefix(r.URL.Path, "/")
	if code == "" || strings.Contains(code, "/") || len(code) > 16 {
		http.NotFound(w, r)
		return
	}
	item, err := h.service.FindByCode(r.Context(), code)
	if errors.Is(err, urlservice.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		h.logger.Error("find short URL failed", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, item.OriginalURL, http.StatusFound)
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (h *Handler) writeError(w http.ResponseWriter, status int, code, message string) {
	response := errorResponse{}
	response.Error.Code = code
	response.Error.Message = message
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func logging(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		logger.Info("request", "method", r.Method, "path", r.URL.Path)
	})
}
