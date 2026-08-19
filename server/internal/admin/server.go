package admin

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/ziomciopoziomcio/digital-music-stand/server/internal/models"
)

//go:embed web/admin.html
var adminHTML embed.FS

type AdminServer struct {
	db   *gorm.DB
	port int
}

func NewAdminServer(db *gorm.DB, port int) *AdminServer {
	return &AdminServer{
		db:   db,
		port: port,
	}
}

func (s *AdminServer) requestAuth(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="Admin Panel restricted access"`)
	http.Error(w, "Unauthorized access", http.StatusUnauthorized)
}

func (s *AdminServer) basicAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		email, password, ok := r.BasicAuth()
		if !ok {
			s.requestAuth(w)
			return
		}

		var user models.User
		if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
			s.requestAuth(w)
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
			s.requestAuth(w)
			return
		}

		if user.Role != "admin" {
			http.Error(w, "Forbidden: Administrator privileges required", http.StatusForbidden)
			return
		}
		if user.Status != "active" {
			http.Error(w, "Forbidden: Admin account is not active", http.StatusForbidden)
			return
		}

		next(w, r)
	}
}

func (s *AdminServer) Start() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/admin", s.basicAuth(func(w http.ResponseWriter, r *http.Request) {
		data, err := adminHTML.ReadFile("web/admin.html")
		if err != nil {
			http.Error(w, "HTML template not found", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
	}))

	mux.HandleFunc("/admin/api/users", s.basicAuth(func(w http.ResponseWriter, r *http.Request) {
		statusFilter := r.URL.Query().Get("status")
		var users []models.User

		query := s.db.Model(&models.User{})
		if statusFilter != "" {
			query = query.Where("status = ?", statusFilter)
		}

		if err := query.Order("id desc").Find(&users).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)
	}))

	mux.HandleFunc("/admin/api/users/approve", s.basicAuth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		idStr := r.URL.Query().Get("id")
		id, _ := strconv.Atoi(idStr)

		if err := s.db.Model(&models.User{}).Where("id = ?", id).Update("status", "active").Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	mux.HandleFunc("/admin/api/users/reject", s.basicAuth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		idStr := r.URL.Query().Get("id")
		id, _ := strconv.Atoi(idStr)

		if err := s.db.Model(&models.User{}).Where("id = ?", id).Update("status", "banned").Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	addr := fmt.Sprintf(":%d", s.port)
	fmt.Printf("Admin Web Panel listening on http://localhost%s/admin\n", addr)
	return http.ListenAndServe(addr, mux)
}
