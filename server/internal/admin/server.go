package admin

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/ziomciopoziomcio/digital-music-stand/server/internal/models"
	"gorm.io/gorm"
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

func (s *AdminServer) Start() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		data, err := adminHTML.ReadFile("web/admin.html")
		if err != nil {
			http.Error(w, "HTML template not found", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
	})

	mux.HandleFunc("/admin/api/users", func(w http.ResponseWriter, r *http.Request) {
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
	})

	mux.HandleFunc("/admin/api/users/approve", func(w http.ResponseWriter, r *http.Request) {
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
	})

	mux.HandleFunc("/admin/api/users/reject", func(w http.ResponseWriter, r *http.Request) {
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
	})

	addr := fmt.Sprintf(":%d", s.port)
	fmt.Printf("Admin Web Panel listening on http://localhost%s/admin\n", addr)
	return http.ListenAndServe(addr, mux)
}
