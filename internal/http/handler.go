package http

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"url-shortener/internal/audit"
	qrservice "url-shortener/internal/qr"
	urlservice "url-shortener/internal/url"
	userservice "url-shortener/internal/user"
)

type Handler struct {
	service       *urlservice.Service
	userService   *userservice.Service
	publicBaseURL string
	cookieSecure  bool
	logger        *slog.Logger
	audit         auditRecorder
	qr            *qrservice.Service
}

type auditRecorder interface {
	Record(context.Context, audit.Event) error
	List(context.Context, int, int) ([]audit.Log, int, error)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	session, ok := h.requireSession(w, r)
	if !ok {
		return
	}
	page, pageSize := pagination(r)
	items, total, err := h.service.List(r.Context(), session.UserID, session.Role == userservice.RoleAdmin, page, pageSize, r.URL.Query().Get("search"), r.URL.Query().Get("status"))
	if err != nil {
		h.logger.Error("list short URLs failed", "error", err)
		h.writeError(w, http.StatusInternalServerError, "LIST_FAILED", "Daftar URL gagal dimuat.")
		return
	}
	response := make([]urlItemResponse, 0, len(items))
	for _, item := range items {
		response = append(response, urlItemResponse{
			ID: item.ID, Code: item.Code, ShortURL: h.publicBaseURL + "/" + item.Code,
			OriginalURL: item.OriginalURL, CreatedAt: item.CreatedAt, Status: item.Status,
			TotalClicks: item.TotalClicks, LastClicked: item.LastClicked,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(urlListResponse{Items: response, Page: page, PageSize: pageSize, TotalItems: total})
}

type urlListResponse struct {
	Items      []urlItemResponse `json:"items"`
	Page       int               `json:"page"`
	PageSize   int               `json:"page_size"`
	TotalItems int               `json:"total_items"`
}

func pagination(r *http.Request) (int, int) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}
	return page, pageSize
}

func (h *Handler) urlAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodDelete && r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	session, ok := h.requireSession(w, r)
	if !ok {
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/urls/")
	isQRPreview := strings.HasSuffix(id, "/qr/preview")
	isQR := isQRPreview || strings.HasSuffix(id, "/qr")
	if id == "" || (strings.Contains(id, "/") && !isQR) {
		http.NotFound(w, r)
		return
	}
	if isQRPreview {
		id = strings.TrimSuffix(id, "/qr/preview")
		if r.Method != http.MethodPut || h.qr == nil || !isUUID(id) {
			http.NotFound(w, r)
			return
		}
	} else if isQR {
		id = strings.TrimSuffix(id, "/qr")
		if (r.Method != http.MethodPut && r.Method != http.MethodGet) || h.qr == nil || !isUUID(id) {
			http.NotFound(w, r)
			return
		}
	} else if r.Method == http.MethodGet && !isUUID(id) {
		http.NotFound(w, r)
		return
	}
	if isQR {
		item, err := h.service.FindByID(r.Context(), id, session.UserID, session.Role == userservice.RoleAdmin)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if r.Method == http.MethodGet {
			saved, err := h.qr.GetConfig(r.Context(), id)
			if errors.Is(err, qrservice.ErrNotFound) {
				if _, genErr := h.qr.Generate(r.Context(), id, h.publicBaseURL+"/"+item.Code, qrservice.Config{}); genErr != nil {
					h.logger.Error("generate missing qr config failed", "error", genErr)
					h.writeError(w, http.StatusInternalServerError, "QR_CONFIG_FAILED", "QR code gagal dimuat.")
					return
				}
				saved, err = h.qr.GetConfig(r.Context(), id)
			}
			if err != nil {
				h.logger.Error("get qr config failed", "error", err)
				h.writeError(w, http.StatusInternalServerError, "QR_CONFIG_FAILED", "QR code gagal dimuat.")
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(qrConfigResponse{
				Foreground: saved.Foreground,
				Background: saved.Background,
				FrameText:  saved.FrameText,
				HasLogo:    saved.HasLogo,
				UpdatedAt:  saved.UpdatedAt,
			})
			return
		}
		if err := r.ParseMultipartForm(5 * 1024 * 1024); err != nil {
			h.writeError(w, http.StatusBadRequest, "INVALID_QR_REQUEST", "File QR tidak valid.")
			return
		}
		file, _, err := r.FormFile("logo")
		var logo []byte
		if err == nil {
			defer file.Close()
			logo, err = io.ReadAll(io.LimitReader(file, 5*1024*1024+1))
		}
		if err != nil && !errors.Is(err, http.ErrMissingFile) {
			h.writeError(w, http.StatusBadRequest, "INVALID_LOGO", "Logo tidak dapat dibaca.")
			return
		}
		config := qrservice.Config{Foreground: r.FormValue("foreground"), Background: r.FormValue("background"), FrameText: r.FormValue("frame_text"), Logo: logo}
		if len(logo) == 0 {
			config.Logo = h.storedLogo(r.Context(), id)
		}
		value := h.publicBaseURL + "/" + item.Code
		var pngData []byte
		if isQRPreview {
			pngData, err = h.qr.Preview(value, config)
		} else {
			pngData, err = h.qr.Generate(r.Context(), id, value, config)
		}
		if err != nil {
			h.writeError(w, http.StatusBadRequest, "INVALID_QR", err.Error())
			return
		}
		w.Header().Set("Content-Type", "image/png")
		if isQRPreview {
			w.Header().Set("Cache-Control", "no-store")
		}
		_, _ = w.Write(pngData)
		return
	}
	if r.Method == http.MethodGet {
		item, err := h.service.FindByID(r.Context(), id, session.UserID, session.Role == userservice.RoleAdmin)
		if errors.Is(err, urlservice.ErrNotFound) {
			http.NotFound(w, r)
			return
		}

		if err != nil {
			h.logger.Error("get short URL failed", "error", err)
			h.writeError(w, http.StatusInternalServerError, "DETAIL_FAILED", "Detail URL gagal dimuat.")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(h.urlResponse(item))
		return
	}
	var err error
	if r.Method == http.MethodPut {
		err = h.service.Enable(r.Context(), id, session.UserID, session.Role == userservice.RoleAdmin)
	} else {
		err = h.service.Disable(r.Context(), id, session.UserID, session.Role == userservice.RoleAdmin)
	}

	if errors.Is(err, urlservice.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		h.logger.Error("disable short URL failed", "error", err)
		h.writeError(w, http.StatusInternalServerError, "UPDATE_FAILED", "Status URL gagal diperbarui.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
	h.recordAudit(r.Context(), audit.Event{ActorUserID: session.UserID, Action: "url.status_changed", ResourceType: "short_url", ResourceID: id, Metadata: map[string]string{"method": r.Method}})
}

func (h *Handler) urlResponse(item urlservice.ShortURL) urlItemResponse {
	return urlItemResponse{
		ID: item.ID, Code: item.Code, ShortURL: h.publicBaseURL + "/" + item.Code,
		OriginalURL: item.OriginalURL, CreatedAt: item.CreatedAt, Status: item.Status,
		TotalClicks: item.TotalClicks, LastClicked: item.LastClicked,
	}
}

func (h *Handler) storedLogo(ctx context.Context, id string) []byte {
	logo, err := h.qr.GetLogo(ctx, id)
	if err != nil && !errors.Is(err, qrservice.ErrNotFound) {
		h.logger.Error("get stored qr logo failed", "url_id", id, "error", err)
		return nil
	}
	return logo
}

func isUUID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for index, char := range value {
		if index == 8 || index == 13 || index == 18 || index == 23 {
			if char != '-' {
				return false
			}
			continue
		}
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return false
		}
	}
	return true
}

type createURLRequest struct {
	OriginalURL string `json:"original_url"`
}

type createURLResponse struct {
	ShortURL string `json:"short_url"`
}

type urlItemResponse struct {
	ID          string     `json:"id"`
	Code        string     `json:"code"`
	ShortURL    string     `json:"short_url"`
	OriginalURL string     `json:"original_url"`
	CreatedAt   time.Time  `json:"created_at"`
	Status      string     `json:"status"`
	TotalClicks int64      `json:"total_clicks"`
	LastClicked *time.Time `json:"last_clicked_at,omitempty"`
}

type qrConfigResponse struct {
	Foreground string    `json:"foreground"`
	Background string    `json:"background"`
	FrameText  string    `json:"frame_text"`
	HasLogo    bool      `json:"has_logo"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type errorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type authResponse struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type userItemResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type userListResponse struct {
	Items      []userItemResponse `json:"items"`
	Page       int                `json:"page"`
	PageSize   int                `json:"page_size"`
	TotalItems int                `json:"total_items"`
}

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func NewHandler(service *urlservice.Service, userService *userservice.Service, publicBaseURL string, logger *slog.Logger) http.Handler {
	return NewHandlerWithOptions(service, userService, publicBaseURL, logger, nil, nil)
}

func NewHandlerWithAudit(service *urlservice.Service, userService *userservice.Service, publicBaseURL string, logger *slog.Logger, audit auditRecorder) http.Handler {
	return NewHandlerWithOptions(service, userService, publicBaseURL, logger, audit, nil)
}

func NewHandlerWithOptions(service *urlservice.Service, userService *userservice.Service, publicBaseURL string, logger *slog.Logger, audit auditRecorder, qr *qrservice.Service) http.Handler {
	return NewHandlerWithCookieSecurity(service, userService, publicBaseURL, logger, audit, qr, false)
}

func NewHandlerWithCookieSecurity(service *urlservice.Service, userService *userservice.Service, publicBaseURL string, logger *slog.Logger, audit auditRecorder, qr *qrservice.Service, cookieSecure bool) http.Handler {
	h := &Handler{
		service:       service,
		userService:   userService,
		publicBaseURL: strings.TrimRight(publicBaseURL, "/"),
		cookieSecure:  cookieSecure,
		logger:        logger,
		audit:         audit,
		qr:            qr,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", h.route)
	mux.HandleFunc("/healthz", h.health)
	mux.HandleFunc("/api/urls", h.create)
	mux.HandleFunc("/api/urls/", h.urlAction)
	mux.HandleFunc("/api/auth/register", h.register)
	mux.HandleFunc("/api/auth/login", h.login)
	mux.HandleFunc("/api/auth/logout", h.logout)
	mux.HandleFunc("/api/auth/me", h.me)
	mux.HandleFunc("/api/users", h.users)
	mux.HandleFunc("/api/users/", h.users)
	mux.HandleFunc("/api/audit-logs", h.auditLogs)
	mux.HandleFunc("/qr/", h.qrImage)
	return securityHeaders(rateLimit(csrf(logging(mux, logger))))
}

func (h *Handler) qrImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet || h.qr == nil {
		http.NotFound(w, r)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/qr/")
	id = strings.TrimSuffix(id, ".png")
	if !isUUID(id) {
		http.NotFound(w, r)
		return
	}
	item, err := h.service.FindPublicByID(r.Context(), id)
	if errors.Is(err, urlservice.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		h.logger.Error("find short URL for qr failed", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	pngData, err := h.qr.Get(r.Context(), id)
	if errors.Is(err, qrservice.ErrNotFound) {
		pngData, err = h.qr.Generate(r.Context(), id, h.publicBaseURL+"/"+item.Code, qrservice.Config{})
	}
	if err != nil {
		h.logger.Error("get qr code failed", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write(pngData)
}

func (h *Handler) auditLogs(w http.ResponseWriter, r *http.Request) {
	session, ok := h.requireSession(w, r)
	if !ok {
		return
	}
	if session.Role != userservice.RoleAdmin || h.audit == nil {
		h.writeError(w, http.StatusForbidden, "FORBIDDEN", "Akses admin diperlukan.")
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	page, pageSize := pagination(r)
	items, total, err := h.audit.List(r.Context(), page, pageSize)
	if err != nil {
		h.logger.Error("list audit logs failed", "error", err)
		h.writeError(w, http.StatusInternalServerError, "AUDIT_LIST_FAILED", "Audit log gagal dimuat.")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Items      []audit.Log `json:"items"`
		Page       int         `json:"page"`
		PageSize   int         `json:"page_size"`
		TotalItems int         `json:"total_items"`
	}{items, page, pageSize, total})
}

func (h *Handler) route(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/assets/") {
		http.StripPrefix("/", http.FileServer(http.Dir("web/dist"))).ServeHTTP(w, r)
		return
	}
	if r.Method == http.MethodGet && isAppPath(r.URL.Path) {
		http.ServeFile(w, r, "web/dist/index.html")
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

func isAppPath(path string) bool {
	return path == "/login" || path == "/dashboard" || path == "/urls" || path == "/users" || path == "/audit" || strings.HasPrefix(path, "/urls/")
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		h.list(w, r)
		return
	}
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
	item, err := h.service.Create(r.Context(), request.OriginalURL, h.userID(r))
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
	if h.qr != nil {
		if _, err := h.qr.Generate(r.Context(), item.ID, h.publicBaseURL+"/"+item.Code, qrservice.Config{}); err != nil {
			h.logger.Error("generate qr code failed", "url_id", item.ID, "error", err)
		}
	}
	h.recordAudit(r.Context(), audit.Event{ActorUserID: h.userID(r), Action: "url.created", ResourceType: "short_url", ResourceID: item.ID})
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var request authRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&request); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Request tidak valid.")
		return
	}
	if _, err := h.userService.Register(r.Context(), request.Email, request.Password); err != nil {
		if errors.Is(err, userservice.ErrInvalidEmail) {
			h.writeError(w, http.StatusBadRequest, "INVALID_CREDENTIALS", "Email atau password tidak valid.")
			return
		}
		h.writeError(w, http.StatusConflict, "REGISTER_FAILED", "Email sudah terdaftar atau tidak dapat digunakan.")
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var request authRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&request); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Request tidak valid.")
		return
	}
	session, err := h.userService.Login(r.Context(), request.Email, request.Password)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Email atau password salah.")
		return
	}
	if err := h.setCSRFToken(w, r, session.ExpiresAt); err != nil {
		h.logger.Error("generate CSRF token failed", "error", err)
		h.writeError(w, http.StatusInternalServerError, "CSRF_TOKEN_FAILED", "Token keamanan gagal dibuat.")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "session", Value: session.Token, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: h.useSecureCookies(r), Expires: session.ExpiresAt})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("session"); err == nil {
		_ = h.userService.Logout(r.Context(), cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: "session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: h.useSecureCookies(r)})
	http.SetCookie(w, &http.Cookie{Name: "csrf", Value: "", Path: "/", MaxAge: -1, SameSite: http.SameSiteLaxMode, Secure: h.useSecureCookies(r)})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Silakan masuk terlebih dahulu.")
		return
	}
	session, err := h.userService.Session(r.Context(), cookie.Value)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak valid.")
		return
	}
	if _, err := r.Cookie("csrf"); err != nil {
		if h.setCSRFToken(w, r, session.ExpiresAt) != nil {
			h.writeError(w, http.StatusInternalServerError, "CSRF_TOKEN_FAILED", "Token keamanan gagal dibuat.")
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(authResponse{Role: session.Role})
}

func (h *Handler) setCSRFToken(w http.ResponseWriter, r *http.Request, expires time.Time) error {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{Name: "csrf", Value: hex.EncodeToString(value), Path: "/", SameSite: http.SameSiteLaxMode, Secure: h.useSecureCookies(r), Expires: expires})
	return nil
}

func (h *Handler) useSecureCookies(r *http.Request) bool {
	return h.cookieSecure || r.TLS != nil
}

func csrf(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions || r.URL.Path == "/api/auth/login" || r.URL.Path == "/api/auth/register" {
			next.ServeHTTP(w, r)
			return
		}
		cookie, err := r.Cookie("csrf")
		header := r.Header.Get("X-CSRF-Token")
		if err != nil || len(cookie.Value) != 64 || len(header) != len(cookie.Value) || subtle.ConstantTimeCompare([]byte(header), []byte(cookie.Value)) != 1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(errorResponse{Error: struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			}{Code: "CSRF_FAILED", Message: "Token keamanan tidak valid."}})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) users(w http.ResponseWriter, r *http.Request) {
	session, ok := h.requireSession(w, r)
	if !ok {
		return
	}
	if session.Role != userservice.RoleAdmin {
		h.writeError(w, http.StatusForbidden, "FORBIDDEN", "Akses admin diperlukan.")
		return
	}
	id := ""
	if r.URL.Path != "/api/users" && r.URL.Path != "/api/users/" {
		id = strings.TrimPrefix(r.URL.Path, "/api/users/")
	}
	if id == "" {
		if r.Method == http.MethodPost {
			h.createManagedUser(w, r)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
	} else if r.Method == http.MethodPatch {
		h.updateManagedUser(w, r, id, session)
		return
	} else if r.Method == http.MethodPut {
		h.updateUserStatus(w, r, id, session)
		return
	} else if r.Method == http.MethodDelete {
		h.deleteManagedUser(w, r, id, session)
		return
	} else {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	page, pageSize := pagination(r)
	items, total, err := h.userService.ListUsers(r.Context(), page, pageSize, strings.TrimSpace(r.URL.Query().Get("search")))
	if err != nil {
		h.logger.Error("list users failed", "error", err)
		h.writeError(w, http.StatusInternalServerError, "LIST_USERS_FAILED", "Daftar user gagal dimuat.")
		return
	}

	response := make([]userItemResponse, 0, len(items))
	for _, item := range items {
		response = append(response, userItemResponse{ID: item.ID, Email: item.Email, Role: item.Role, Status: item.Status, CreatedAt: item.CreatedAt})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(userListResponse{Items: response, Page: page, PageSize: pageSize, TotalItems: total})
}

func (h *Handler) createManagedUser(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&request); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Request tidak valid.")
		return
	}
	item, err := h.userService.CreateManagedUser(r.Context(), request.Email, request.Password, request.Role)
	if err != nil {
		if errors.Is(err, userservice.ErrInvalidEmail) || errors.Is(err, userservice.ErrInvalidRole) {
			h.writeError(w, http.StatusBadRequest, "INVALID_USER", "Data user tidak valid.")
			return
		}
		h.writeError(w, http.StatusConflict, "CREATE_USER_FAILED", "User gagal dibuat.")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(userItemResponse{ID: item.ID, Email: item.Email, Role: item.Role, Status: item.Status, CreatedAt: item.CreatedAt})
	h.recordAudit(r.Context(), audit.Event{ActorUserID: h.userID(r), Action: "user.created", ResourceType: "user", ResourceID: item.ID, Metadata: map[string]string{"role": item.Role}})
}

func (h *Handler) updateManagedUser(w http.ResponseWriter, r *http.Request, id string, session userservice.Session) {
	if strings.Contains(id, "/") {
		http.NotFound(w, r)
		return
	}
	var request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2048)).Decode(&request); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Request tidak valid.")
		return
	}
	if id == session.UserID && request.Role != userservice.RoleAdmin {
		h.writeError(w, http.StatusBadRequest, "SELF_DEMOTION", "Admin tidak dapat menurunkan role dirinya sendiri.")
		return
	}
	if err := h.userService.UpdateUser(r.Context(), id, request.Email, request.Password, request.Role); err != nil {
		if errors.Is(err, userservice.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		if errors.Is(err, userservice.ErrInvalidEmail) || errors.Is(err, userservice.ErrInvalidRole) {
			h.writeError(w, http.StatusBadRequest, "INVALID_USER", "Data user tidak valid.")
			return
		}
		h.logger.Error("update user failed", "error", err)
		h.writeError(w, http.StatusInternalServerError, "UPDATE_USER_FAILED", "Role user gagal diperbarui.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
	h.recordAudit(r.Context(), audit.Event{ActorUserID: session.UserID, Action: "user.updated", ResourceType: "user", ResourceID: id})
}

func (h *Handler) updateUserStatus(w http.ResponseWriter, r *http.Request, id string, session userservice.Session) {
	if id == session.UserID {
		h.writeError(w, http.StatusBadRequest, "SELF_DEACTIVATION", "Admin tidak dapat menonaktifkan dirinya sendiri.")
		return
	}
	var request struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 512)).Decode(&request); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Request tidak valid.")
		return
	}
	if err := h.userService.UpdateStatus(r.Context(), id, request.Status); err != nil {
		if errors.Is(err, userservice.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		h.writeError(w, http.StatusBadRequest, "INVALID_STATUS", "Status user tidak valid.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
	h.recordAudit(r.Context(), audit.Event{ActorUserID: session.UserID, Action: "user.status_changed", ResourceType: "user", ResourceID: id, Metadata: map[string]string{"status": request.Status}})
}

func (h *Handler) deleteManagedUser(w http.ResponseWriter, r *http.Request, id string, session userservice.Session) {
	if id == session.UserID {
		h.writeError(w, http.StatusBadRequest, "SELF_DELETE", "Admin tidak dapat menghapus dirinya sendiri.")
		return
	}
	if err := h.userService.DeleteUser(r.Context(), id); err != nil {
		if errors.Is(err, userservice.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		h.logger.Error("delete user failed", "error", err)
		h.writeError(w, http.StatusInternalServerError, "DELETE_USER_FAILED", "User gagal dihapus.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
	h.recordAudit(r.Context(), audit.Event{ActorUserID: session.UserID, Action: "user.deleted", ResourceType: "user", ResourceID: id})
}

func (h *Handler) userID(r *http.Request) string {
	session, ok := h.session(r)
	if !ok {
		return ""
	}
	return session.UserID
}

func (h *Handler) session(r *http.Request) (userservice.Session, bool) {
	cookie, err := r.Cookie("session")
	if err != nil || h.userService == nil {
		return userservice.Session{}, false
	}
	session, err := h.userService.Session(r.Context(), cookie.Value)
	return session, err == nil
}

func (h *Handler) requireSession(w http.ResponseWriter, r *http.Request) (userservice.Session, bool) {
	session, ok := h.session(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Silakan masuk terlebih dahulu.")
	}
	return session, ok
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
	if err := h.service.RecordClick(r.Context(), item.ID, r.UserAgent(), r.Referer()); err != nil {
		h.logger.Error("record URL click failed", "url_id", item.ID, "error", err)
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

func (h *Handler) recordAudit(ctx context.Context, event audit.Event) {
	if h.audit == nil {
		return
	}
	if err := h.audit.Record(ctx, event); err != nil {
		h.logger.Error("audit log failed", "action", event.Action, "error", err)
	}
}

type rateLimiter struct {
	mu      sync.Mutex
	windows map[string]rateWindow
}

type rateWindow struct {
	start time.Time
	count int
}

func rateLimit(next http.Handler) http.Handler {
	limiter := &rateLimiter{windows: make(map[string]rateWindow)}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.RemoteAddr
		if host, _, splitErr := strings.Cut(r.RemoteAddr, ":"); splitErr {
			key = host
		}
		limit := 120
		bucket := "default"
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/auth/"):
			limit = 20
		case strings.HasSuffix(r.URL.Path, "/qr/preview"):
			bucket = "qr-preview"
			limit = 300
		}
		key += "|" + bucket
		now := time.Now()
		limiter.mu.Lock()
		window := limiter.windows[key]
		if window.start.IsZero() || now.Sub(window.start) >= time.Minute {
			window = rateWindow{start: now}
		}
		window.count++
		limiter.windows[key] = window
		limited := window.count > limit
		limiter.mu.Unlock()
		if limited {
			w.Header().Set("Retry-After", "60")
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self';")
		next.ServeHTTP(w, r)
	})
}
