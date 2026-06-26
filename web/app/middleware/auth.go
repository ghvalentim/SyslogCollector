package auth

import (
	"net/http"
	"time"
	SQL "syslog-web/data/SQL"
	 helper "syslog-web/app/helpers"
)

// Utiliza um token simples baseado em Cookies para gestão de sessão
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		helper.RenderTemplate(w, "views/login.html", nil)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	var dbPass string
	// Agora procuramos na nova tabela de utilizadores em vez da tabela settings
	err := SQL.DB.QueryRow("SELECT password_hash FROM users WHERE username = $1", username).Scan(&dbPass)

	// NOTA: Usa comparação direta pois migrámos a password legada.
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

	helper.RenderTemplate(w, "views/login.html", map[string]string{"Error": "Credenciais incorretas ou utilizador inexistente."})
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:    "syslog_session",
		Value:   "",
		Expires: time.Now().Add(-1 * time.Hour),
		Path:    "/",
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("syslog_session")
		if err != nil || cookie.Value == "" {
			if r.Header.Get("HX-Request") == "true" {
				w.WriteHeader(http.StatusUnauthorized)
			} else {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
			}
			return
		}
		next.ServeHTTP(w, r)
	}
}

// ServeStaticFiles serve ficheiros estáticos (CSS, JS, imagens) a partir do diretório "assets".
func ServeDashboard(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, "index.html") }
func ServeScript(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, "assets/script.js") }
func ServeStyles(w http.ResponseWriter, r *http.Request) { w.Header().Set("Content-Type", "text/css"); http.ServeFile(w, r, "assets/output.css") }