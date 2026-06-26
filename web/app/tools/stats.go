package tools

import (
	"net/http"
	_ "github.com/redis/go-redis/v9"
	_ "github.com/lib/pq"
	"encoding/json"
	"syslog-web/data/SQL"
	"syslog-web/models"
	"syslog-web/app/helpers"
)


// --- ESTATÍSTICAS ---
// serveStatsView renderiza a página de estatísticas do painel de administração.
func ServeStatsView(w http.ResponseWriter, r *http.Request) { helpers.RenderTemplate(w, "views/stats.html", nil) }

// fetchStatsData obtém estatísticas agregadas da base de dados e retorna como JSON para a interface do utilizador.
func FetchStatsData(w http.ResponseWriter, r *http.Request) {
	var res models.StatsResponse
	rows, _ := SQL.DB.Query("SELECT severity, COUNT(*) FROM syslogs GROUP BY severity ORDER BY count DESC"); defer rows.Close()
	for rows.Next() { var s models.SeverityStat; rows.Scan(&s.Severity, &s.Count); res.Severities = append(res.Severities, s) }
	rows2, _ := SQL.DB.Query("SELECT hostname, COUNT(*) FROM syslogs WHERE hostname != '-' GROUP BY hostname ORDER BY count DESC LIMIT 5"); defer rows2.Close()
	for rows2.Next() { var h models.HostStat; rows2.Scan(&h.Hostname, &h.Count); res.Hosts = append(res.Hosts, h) }
	rows3, _ := SQL.DB.Query("SELECT source_type, COUNT(*) FROM syslogs GROUP BY source_type ORDER BY count DESC"); defer rows3.Close()
	for rows3.Next() { var src models.SourceStat; rows3.Scan(&src.Source, &src.Count); res.Sources = append(res.Sources, src) }
	w.Header().Set("Content-Type", "application/json"); json.NewEncoder(w).Encode(res)
}