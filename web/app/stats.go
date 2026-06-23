package app

import (
	"net/http"
	_ "github.com/redis/go-redis/v9"
	_ "github.com/lib/pq"
	"encoding/json"
	"syslog-web/database"
	"syslog-web/models"
)


// --- ESTATÍSTICAS ---
func serveStatsView(w http.ResponseWriter, r *http.Request) { RenderTemplate(w, "templates/stats.html", nil) }

func fetchStatsData(w http.ResponseWriter, r *http.Request) {
	var res model.StatsResponse
	rows, _ := database.DB.Query("SELECT severity, COUNT(*) FROM syslogs GROUP BY severity ORDER BY count DESC"); defer rows.Close()
	for rows.Next() { var s model.SeverityStat; rows.Scan(&s.Severity, &s.Count); res.Severities = append(res.Severities, s) }
	rows2, _ := database.DB.Query("SELECT hostname, COUNT(*) FROM syslogs WHERE hostname != '-' GROUP BY hostname ORDER BY count DESC LIMIT 5"); defer rows2.Close()
	for rows2.Next() { var h model.HostStat; rows2.Scan(&h.Hostname, &h.Count); res.Hosts = append(res.Hosts, h) }
	rows3, _ := database.DB.Query("SELECT source_type, COUNT(*) FROM syslogs GROUP BY source_type ORDER BY count DESC"); defer rows3.Close()
	for rows3.Next() { var src model.SourceStat; rows3.Scan(&src.Source, &src.Count); res.Sources = append(res.Sources, src) }
	w.Header().Set("Content-Type", "application/json"); json.NewEncoder(w).Encode(res)
}