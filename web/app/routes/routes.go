package routes

import (
	"net/http"
	"syslog-web/api"
	"syslog-web/app/middleware"
	"syslog-web/app/settings"
	"syslog-web/app/tools"
	"syslog-web/app/entities"
	"syslog-web/app/logs"
)
	

func InitRoutes() {
	RegisterRoutes()
}
func RegisterRoutes() {
	// Recursos estáticos
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))
	http.Handle("/output.css", http.FileServer(http.Dir(".")))
	
	// Autenticação
	http.HandleFunc("/login", auth.LoginHandler)
	http.HandleFunc("/logout", auth.LogoutHandler)

	// Rotas Protegidas (Views)
	http.HandleFunc("/", auth.AuthMiddleware(auth.ServeDashboard))
	http.HandleFunc("/logs", auth.AuthMiddleware(logs.ServeLogsPartial))
	http.HandleFunc("/stats/view", auth.AuthMiddleware(tools.ServeStatsView))
	http.HandleFunc("/api/stats", auth.AuthMiddleware(api.APIStats))
	http.HandleFunc("/settings/view", auth.AuthMiddleware(settings.ServeSettingsView))
	http.HandleFunc("/settings/save", auth.AuthMiddleware(settings.SaveSettings))
	http.HandleFunc("/tools/view", auth.AuthMiddleware(settings.ServeToolsView))
	http.HandleFunc("/policies/view", auth.AuthMiddleware(settings.ServePoliciesView))
	http.HandleFunc("/policies/save", auth.AuthMiddleware(settings.SavePolicies))
	http.HandleFunc("/alerts/view", auth.AuthMiddleware(tools.ServeAlertsView))
	http.HandleFunc("/alerts/save", auth.AuthMiddleware(tools.SaveAlertRule))
	http.HandleFunc("/alerts/delete", auth.AuthMiddleware(tools.DeleteAlertRule))
	
	// NOVO: Rotas de Gestão de Utilizadores
	http.HandleFunc("/users/save", auth.AuthMiddleware(entities.SaveUser))
	http.HandleFunc("/users/delete", auth.AuthMiddleware(entities.DeleteUser))
}