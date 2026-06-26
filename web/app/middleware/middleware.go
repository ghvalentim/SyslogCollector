package middleware

import (
	"net/http"
	"time"
	SQL "syslog-web/data/SQL"
	 helper "syslog-web/app/helpers"
)

type AuthMiddleware struct {
	Next http.HandlerFunc
	Request *http.Request
	Writer http.ResponseWriter
}


func (am *AuthMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	am.Request = r
	am.Writer = w

	cookie, err := r.Cookie("syslog_session")
	if err != nil || cookie.Value == "" {
		if r.Header.Get("HX-Request") == "true" {
			w.WriteHeader(http.StatusUnauthorized)
		} else {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
		}
		return
	}

	am.Next.ServeHTTP(w, r)
}

func NewAuthMiddleware(next http.HandlerFunc) *AuthMiddleware {
	return &AuthMiddleware{
		Next: next,
	}
}

func (am *AuthMiddleware) Middleware() *AuthMiddleware {
	return am
}

func (am *AuthMiddleware) Path() string {
	return am.Request.URL.Path
}

func (am *AuthMiddleware) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		am.RenderTemplate(w, r, "views/login.html", nil)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	var dbPass string
	err := SQL.DB.QueryRow("SELECT password_hash FROM users WHERE username = $1", username).Scan(&dbPass)

	if err == nil && password == dbPass {
		http.SetCookie(w, &http.Cookie{
			Name:     "syslog_session",
			Value:    username,
			Expires:  time.Now().Add(24 * time.Hour),
			HttpOnly: true,
			Path:     "/",
		})
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	am.RenderTemplate(w, r, "views/login.html", map[string]string{"Error": "Credenciais incorretas ou utilizador inexistente."})
}

func (am *AuthMiddleware) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:    "syslog_session",
		Value:   "",
		Expires: time.Now().Add(-1 * time.Hour),
		Path:    "/",
	})
}

func (am *AuthMiddleware) IsAuthenticated() bool {
	cookie, err := am.Request.Cookie("syslog_session")
	if err != nil || cookie.Value == "" {
		return false
	}
	return true
}

func (am *AuthMiddleware) ServeDashboard(w http.ResponseWriter, r *http.Request) {
	am.RenderTemplate(w, r, "views/dashboard.html", nil)
}

func (am *AuthMiddleware) ServeStatsView(w http.ResponseWriter, r *http.Request) {
	am.RenderTemplate(w, r, "views/stats.html", nil)
}

func (am *AuthMiddleware) ServeStyles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css")
	http.ServeFile(w, r, "assets/output.css")
}

func (am *AuthMiddleware) RenderTemplate(w http.ResponseWriter, r *http.Request, templateName string, data interface{}) {
	helper.RenderTemplate(w, templateName, data)
}
