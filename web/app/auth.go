package app

import (
	"net/http"
	"html/template"
	_ "github.com/lib/pq"
	"log"
	"syslog-web/database"
	"syslog-web/models"
)


// --- MIDDLEWARES & RENDERS ---
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		validPaths := map[string]bool{ "/": true, "/logs": true, "/export": true, "/stats/view": true, "/api/stats": true, "/settings/view": true, "/settings/save": true, "/tools/view": true, "/tools/download": true, "/policies/view": true, "/policies/save": true, "/alerts/view": true, "/alerts/save": true }
		if !validPaths[r.URL.Path] { http.NotFound(w, r); return }

		cookie, err := r.Cookie("admin_session")
		if err != nil || cookie.Value != "valid" {
			if r.Header.Get("HX-Request") == "true" {
				w.Header().Set("HX-Redirect", "/login")
				w.WriteHeader(http.StatusUnauthorized)
			} else {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
			}
			return
		}
		next.ServeHTTP(w, r)
	}
}


func serveLogin(w http.ResponseWriter, r *http.Request) {
	var s models.Settings
	if r.Method == "POST" {
		user, pass := r.FormValue("username"), r.FormValue("password")
		var DBUser, DBPass string
		database.DB.QueryRow("SELECT admin_user, admin_pass FROM settings WHERE id = 1").Scan(&DBUser, &DBPass)

		if user == DBUser && pass == DBPass {
			http.SetCookie(w, &http.Cookie{Name: "admin_session", Value: "valid", Path: "/", HttpOnly: true, MaxAge: 86400})
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		s.Error = "Credenciais incorretas!"
	}
	// Carrega de ficheiro em vez de ter HTML inline!
	RenderTemplate(w, "templates/login.html", s)
}

func handleLogout(w http.ResponseWriter, r *http.Request) { http.SetCookie(w, &http.Cookie{Name: "admin_session", Value: "", Path: "/", MaxAge: -1}); http.Redirect(w, r, "/login", http.StatusSeeOther) }


// Helper Inteligente: Carrega os ficheiros HTML externos!
func RenderTemplate(w http.ResponseWriter, path string, data interface{}) {
	t, err := template.ParseFiles(path)
	if err != nil {
		log.Printf("[ERRO] Não foi possível carregar o template %s: %v", path, err)
		http.Error(w, "Erro interno de apresentação visual.", http.StatusInternalServerError)
		return
	}
	t.Execute(w, data)
}

func serveDashboard(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, "index.html") }
func serveScript(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, "assets/script.js") }
func serveStyles(w http.ResponseWriter, r *http.Request) { w.Header().Set("Content-Type", "text/css"); http.ServeFile(w, r, "assets/output.css") }