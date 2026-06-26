package middleware

import (
	"net/http"
	"log"
)

type Adapter func(http.Handler) http.Handler

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[REQUEST] %s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[PANIC RECOVERED] Erro Crítico: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := getSession(r)
		if session == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

func getSession(r *http.Request) (string) { 
	getSessionCookie, err := r.Cookie("session_id")
	if err != nil {
		return ""
	}
	return getSessionCookie.Value
}
