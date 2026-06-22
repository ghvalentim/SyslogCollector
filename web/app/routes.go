package app

import (
	"net/http"
	_ "github.com/redis/go-redis/v9"
	_ "github.com/lib/pq"
	"log"
)

func InitRoutes() {

// Rotas Públicas e Ficheiros Estáticos
	
	http.HandleFunc("/login", serveLogin)
	http.HandleFunc("/logout", handleLogout)
	http.HandleFunc("/assets/script.js", serveScript)
	http.HandleFunc("/assets/output.css", serveStyles)

	// Rotas Protegidas (Requerem Autenticação)
	http.HandleFunc("/", AuthMiddleware(serveDashboard))
	http.HandleFunc("/logs", AuthMiddleware(fetchLogsHTML))
	http.HandleFunc("/export", AuthMiddleware(exportCSV))
	http.HandleFunc("/stats/view", AuthMiddleware(serveStatsView))
	http.HandleFunc("/api/stats", AuthMiddleware(fetchStatsData))
	http.HandleFunc("/settings/view", AuthMiddleware(ServeSettingsView))
	http.HandleFunc("/settings/save", AuthMiddleware(SaveSettings))
	http.HandleFunc("/tools/view", AuthMiddleware(ServeToolsView))
	http.HandleFunc("/tools/download", AuthMiddleware(DownloadTool))
	http.HandleFunc("/policies/view", AuthMiddleware(servePoliciesView))
	http.HandleFunc("/policies/save", AuthMiddleware(savePolicies))
	http.HandleFunc("/alerts/view", AuthMiddleware(ServeAlertsView))
	http.HandleFunc("/alerts/save", AuthMiddleware(SaveAlertRule))
	http.HandleFunc("/alerts/delete", AuthMiddleware(DeleteAlertRule))

	log.Println("Painel de Administração ativo na porta 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Erro HTTP: %v", err)
	}

}