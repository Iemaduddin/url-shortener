package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	urlservice "url-shortener/internal/url"
	userservice "url-shortener/internal/user"
)

type Handler struct {
	service       *urlservice.Service
	userService   *userservice.Service
	publicBaseURL string
	logger        *slog.Logger
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	session, ok := h.requireSession(w, r)
	if !ok {
		return
	}
	items, err := h.service.List(r.Context(), session.UserID, session.Role == userservice.RoleAdmin)
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
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func (h *Handler) urlAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete && r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	session, ok := h.requireSession(w, r)
	if !ok {
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/urls/")
	if id == "" || strings.Contains(id, "/") {
		http.NotFound(w, r)
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
}

type createURLRequest struct {
	OriginalURL string `json:"original_url"`
}

type createURLResponse struct {
	ShortURL string `json:"short_url"`
}

type urlItemResponse struct {
	ID          string    `json:"id"`
	Code        string    `json:"code"`
	ShortURL    string    `json:"short_url"`
	OriginalURL string    `json:"original_url"`
	CreatedAt   time.Time `json:"created_at"`
	Status      string    `json:"status"`
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

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func NewHandler(service *urlservice.Service, userService *userservice.Service, publicBaseURL string, logger *slog.Logger) http.Handler {
	h := &Handler{
		service:       service,
		userService:   userService,
		publicBaseURL: strings.TrimRight(publicBaseURL, "/"),
		logger:        logger,
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
	return logging(mux, logger)
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
	return path == "/dashboard" || path == "/urls" || path == "/users"
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
	http.SetCookie(w, &http.Cookie{Name: "session", Value: session.Token, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: r.TLS != nil, Expires: session.ExpiresAt})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("session"); err == nil {
		_ = h.userService.Logout(r.Context(), cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: "session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode})
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
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(authResponse{Role: session.Role})
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
	items, err := h.userService.ListUsers(r.Context())
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
	_ = json.NewEncoder(w).Encode(response)
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
