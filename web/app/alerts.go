package app

import (
	"net/http"
	_ "github.com/lib/pq"
)

// --- ALERTAS ---
func serveAlertsView(w http.ResponseWriter, r *http.Request) {
	var rules []AlertRule
	rows, _ := db.Query("SELECT id, enabled, name, severity, source_type, keyword, threshold, window_minutes FROM alert_rules ORDER BY id DESC")
	defer rows.Close()
	for rows.Next() { var ar AlertRule; rows.Scan(&ar.ID, &ar.Enabled, &ar.Name, &ar.Severity, &ar.SourceType, &ar.Keyword, &ar.Threshold, &ar.WindowMinutes); rules = append(rules, ar) }
	RenderTemplate(w, "templates/alerts.html", rules)
}

func saveAlertRule(w http.ResponseWriter, r *http.Request) {
	db.Exec("INSERT INTO alert_rules (name, source_type, keyword, threshold, window_minutes) VALUES ($1, $2, $3, $4, $5)", r.FormValue("name"), r.FormValue("source_type"), r.FormValue("keyword"), r.FormValue("threshold"), r.FormValue("window"))
	w.Write([]byte(`<div class="p-3 bg-emerald-50 text-emerald-700 rounded-lg border border-emerald-200 text-sm flex items-center font-medium"><i data-lucide="check-circle" class="w-5 h-5 mr-2"></i> Regra base gravada. (Monitorização agendada para a próxima Sprint)</div><script>lucide.createIcons();</script>`))
}