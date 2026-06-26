package tools

import (
	"net/http"
	"fmt"
	"time"
	"encoding/csv"
	"syslog-web/models"
	SQL "syslog-web/data/SQL"
	"syslog-web/app/helpers"
	_ "github.com/redis/go-redis/v9"
	_ "github.com/lib/pq"
)


// --- LOGS ---
// buildLogsQuery constrói a query SQL para buscar logs com base nos parâmetros da requisição.
func buildLogsQuery(r *http.Request, limit int) (string, []interface{}) {
	q, sev, source := r.URL.Query().Get("q"), r.URL.Query().Get("sev"), r.URL.Query().Get("source")
	query := "SELECT id, timestamp, source_ip, protocol, hostname, app_name, severity, facility, facility_name, source_type, payload FROM syslogs WHERE 1=1"
	var args []interface{}; argId := 1

	if q != "" { query += fmt.Sprintf(" AND (source_ip ILIKE $%d OR payload ILIKE $%d OR hostname ILIKE $%d OR app_name ILIKE $%d)", argId, argId, argId, argId); args = append(args, "%"+q+"%"); argId++ }
	if sev != "" { query += fmt.Sprintf(" AND severity = $%d", argId); args = append(args, sev); argId++ }
	if source != "" { query += fmt.Sprintf(" AND source_type = $%d", argId); args = append(args, source); argId++ }
	
	query += " ORDER BY timestamp DESC"
	if limit > 0 { query += fmt.Sprintf(" LIMIT %d", limit) }
	return query, args
}

// fetchLogsHTML busca logs da base de dados e renderiza a página HTML com os resultados.
func fetchLogsHTML(w http.ResponseWriter, r *http.Request) {
	query, args := buildLogsQuery(r, 50)
	rows, err := SQL.DB.Query(query, args...)
	if err != nil { http.Error(w, "Erro", http.StatusInternalServerError); return }
	defer rows.Close()

	var logs []models.LogEntry
	for rows.Next() {
		var l models.LogEntry; var ts time.Time
		if rows.Scan(&l.ID, &ts, &l.SourceIP, &l.Protocol, &l.Hostname, &l.AppName, &l.Severity, &l.Facility, &l.FacilityName, &l.SourceType, &l.Payload) == nil {
			l.Timestamp = ts.Format("2006-01-02 15:04:05"); logs = append(logs, l)
		}
	}
	helpers.RenderTemplate(w, "views/logs.html", logs)
}

// exportCSV exporta logs da base de dados para um ficheiro CSV e envia como resposta HTTP.
func exportCSV(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment;filename=syslogs.csv")
	w.Write([]byte{0xEF, 0xBB, 0xBF})
	writer := csv.NewWriter(w); defer writer.Flush()
	writer.Write([]string{"ID", "Data", "Origem IP", "Host", "App", "Source Type", "Facility", "Gravidade", "Msg"})
	query, args := buildLogsQuery(r, 2000); rows, _ := SQL.DB.Query(query, args...); defer rows.Close()
	for rows.Next() { var l models.LogEntry; var ts time.Time; if rows.Scan(&l.ID, &ts, &l.SourceIP, &l.Protocol, &l.Hostname, &l.AppName, &l.Severity, &l.Facility, &l.FacilityName, &l.SourceType, &l.Payload) == nil { writer.Write([]string{fmt.Sprint(l.ID), ts.Format("2006-01-02 15:04:05"), l.SourceIP, l.Hostname, l.AppName, l.SourceType, l.FacilityName, l.Severity, l.Payload}) } }
}

// GetUserMail retorna o email do administrador configurado nas definições da aplicação.
func GetUserMail() string {
	var email string
	SQL.DB.QueryRow("SELECT admin_email FROM settings WHERE id = 1").Scan(&email)
	return email
}
