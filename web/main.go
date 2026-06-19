package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

// --- ESTRUTURAS DE DADOS ---
type LogEntry struct {
	ID           int    `json:"id"`
	Timestamp    string `json:"timestamp"`
	SourceIP     string `json:"source_ip"`
	Protocol     string `json:"protocol"`
	Hostname     string `json:"hostname"`
	AppName      string `json:"app_name"`
	Severity     string `json:"severity"`
	Facility     string `json:"facility"`
	FacilityName string `json:"facility_name"`
	SourceType   string `json:"source_type"`
	Payload      string `json:"payload"`
}

type SeverityStat struct { Severity string `json:"severity"`; Count int `json:"count"` }
type HostStat struct { Hostname string `json:"hostname"`; Count int `json:"count"` }
type SourceStat struct { Source string `json:"source"`; Count int `json:"count"` }
type StatsResponse struct { Severities []SeverityStat `json:"severities"`; Hosts []HostStat `json:"hosts"`; Sources []SourceStat `json:"sources"` }
type Settings struct { Retention int; User string; Error string }

type LogPolicy struct {
	Enabled         bool     `json:"enabled"`
	MinimumSeverity string   `json:"minimum_severity"`
	IgnoredApps     []string `json:"ignored_apps"`
	IgnoredHosts    []string `json:"ignored_hosts"`
	IgnoredKeywords []string `json:"ignored_keywords"`
}

type PolicyViewData struct {
	Policy      LogPolicy
	AppsStr     string
	HostsStr    string
	KeywordsStr string
	Stats       map[string]string
}

type AlertRule struct {
	ID            int
	Enabled       bool
	Name          string
	Severity      string
	SourceType    string
	Keyword       string
	Threshold     int
	WindowMinutes int
}

var (
	db  *sql.DB
	rdb *redis.Client
	ctx = context.Background()
)

func main() {
	defer func() { if r := recover(); r != nil { log.Fatalf("[CRÍTICO] Falha fatal: %v", r) } }()

	initRedis()
	initDB()

	go logWorker()
	go retentionWorker()

	// Rotas Públicas e Ficheiros Estáticos
	http.HandleFunc("/login", serveLogin)
	http.HandleFunc("/logout", handleLogout)
	http.HandleFunc("/script.js", serveScript)
	http.HandleFunc("/output.css", serveStyles)

	// Rotas Protegidas (Requerem Autenticação)
	http.HandleFunc("/", authMiddleware(serveDashboard))
	http.HandleFunc("/logs", authMiddleware(fetchLogsHTML))
	http.HandleFunc("/export", authMiddleware(exportCSV))
	http.HandleFunc("/stats/view", authMiddleware(serveStatsView))
	http.HandleFunc("/api/stats", authMiddleware(fetchStatsData))
	http.HandleFunc("/settings/view", authMiddleware(serveSettingsView))
	http.HandleFunc("/settings/save", authMiddleware(saveSettings))
	http.HandleFunc("/tools/view", authMiddleware(serveToolsView))
	http.HandleFunc("/tools/download", authMiddleware(downloadTool))
	http.HandleFunc("/policies/view", authMiddleware(servePoliciesView))
	http.HandleFunc("/policies/save", authMiddleware(savePolicies))
	http.HandleFunc("/alerts/view", authMiddleware(serveAlertsView))
	http.HandleFunc("/alerts/save", authMiddleware(saveAlertRule))

	log.Println("Painel de Administração ativo na porta 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Erro HTTP: %v", err)
	}
}

// --- MIDDLEWARES & RENDERS ---
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
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

// Helper Inteligente: Carrega os ficheiros HTML externos!
func renderTemplate(w http.ResponseWriter, path string, data interface{}) {
	t, err := template.ParseFiles(path)
	if err != nil {
		log.Printf("[ERRO] Não foi possível carregar o template %s: %v", path, err)
		http.Error(w, "Erro interno de apresentação visual.", http.StatusInternalServerError)
		return
	}
	t.Execute(w, data)
}

func serveDashboard(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, "index.html") }
func serveScript(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, "script.js") }
func serveStyles(w http.ResponseWriter, r *http.Request) { w.Header().Set("Content-Type", "text/css"); http.ServeFile(w, r, "output.css") }
func handleLogout(w http.ResponseWriter, r *http.Request) { http.SetCookie(w, &http.Cookie{Name: "admin_session", Value: "", Path: "/", MaxAge: -1}); http.Redirect(w, r, "/login", http.StatusSeeOther) }

// --- LOGIN ---
func serveLogin(w http.ResponseWriter, r *http.Request) {
	var s Settings
	if r.Method == "POST" {
		user, pass := r.FormValue("username"), r.FormValue("password")
		var dbUser, dbPass string
		db.QueryRow("SELECT admin_user, admin_pass FROM settings WHERE id = 1").Scan(&dbUser, &dbPass)

		if user == dbUser && pass == dbPass {
			http.SetCookie(w, &http.Cookie{Name: "admin_session", Value: "valid", Path: "/", HttpOnly: true, MaxAge: 86400})
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		s.Error = "Credenciais incorretas!"
	}
	// Carrega de ficheiro em vez de ter HTML inline!
	renderTemplate(w, "templates/login.html", s)
}

// --- INIT (DB & REDIS) ---
func initRedis() {
	rdb = redis.NewClient(&redis.Options{Addr: os.Getenv("REDIS_URL"), DB: 0})
	if _, err := rdb.Ping(ctx).Result(); err != nil { log.Fatalf("Erro Redis: %v", err) }
}

func initDB() {
	connStr := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable", os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASS"), os.Getenv("DB_NAME"))
	for i := 0; i < 5; i++ { db, _ = sql.Open("postgres", connStr); if db.Ping() == nil { break }; time.Sleep(3 * time.Second) }
	
	db.Exec(`CREATE TABLE IF NOT EXISTS syslogs (id SERIAL PRIMARY KEY, timestamp TIMESTAMP, source_ip VARCHAR(50), protocol VARCHAR(10), hostname VARCHAR(100) DEFAULT '-', app_name VARCHAR(100) DEFAULT '-', severity VARCHAR(20) DEFAULT 'Info', facility VARCHAR(100) DEFAULT '-', facility_name VARCHAR(50) DEFAULT 'Unknown', source_type VARCHAR(50) DEFAULT 'Unknown', payload TEXT, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP); CREATE INDEX IF NOT EXISTS idx_timestamp ON syslogs(timestamp DESC);`)
	db.Exec("ALTER TABLE syslogs ADD COLUMN IF NOT EXISTS source_type VARCHAR(50) DEFAULT 'Unknown'")
	db.Exec("ALTER TABLE syslogs ADD COLUMN IF NOT EXISTS facility_name VARCHAR(50) DEFAULT 'Unknown'")

	db.Exec(`CREATE TABLE IF NOT EXISTS settings (id SERIAL PRIMARY KEY, retention_days INT DEFAULT 30, admin_user VARCHAR(100) DEFAULT 'admin', admin_pass VARCHAR(100) DEFAULT 'admin');`)
	db.Exec(`INSERT INTO settings (id) SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM settings WHERE id = 1)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS log_policies (id SERIAL PRIMARY KEY, enabled BOOLEAN DEFAULT false, minimum_severity VARCHAR(20) DEFAULT 'Info', ignored_apps TEXT DEFAULT '', ignored_hosts TEXT DEFAULT '', ignored_keywords TEXT DEFAULT '');`)
	db.Exec(`INSERT INTO log_policies (id) SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM log_policies WHERE id = 1)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS alert_rules (id SERIAL PRIMARY KEY, enabled BOOLEAN DEFAULT true, name VARCHAR(100), severity VARCHAR(20), source_type VARCHAR(50), keyword VARCHAR(100), threshold INT, window_minutes INT, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP);`)

	syncPolicyToRedis()
}

func syncPolicyToRedis() {
	var enabled bool; var minSev, appsStr, hostsStr, kwStr string
	db.QueryRow("SELECT enabled, minimum_severity, ignored_apps, ignored_hosts, ignored_keywords FROM log_policies WHERE id=1").Scan(&enabled, &minSev, &appsStr, &hostsStr, &kwStr)
	policy := LogPolicy{ Enabled: enabled, MinimumSeverity: minSev, IgnoredApps: parseList(appsStr), IgnoredHosts: parseList(hostsStr), IgnoredKeywords: parseList(kwStr) }
	jsonData, _ := json.Marshal(policy)
	rdb.Set(ctx, "active_log_policy", jsonData, 0)
	rdb.Publish(ctx, "policy_updates", "reload")
}

func parseList(i string) []string { p := strings.Split(i, ","); var r []string; for _, x := range p { if t := strings.TrimSpace(x); t != "" { r = append(r, t) } }; return r }

// --- WORKERS ---
func logWorker() {
	defer func() { if r := recover(); r != nil { time.Sleep(2 * time.Second); go logWorker() } }()
	for {
		res, err := rdb.BRPop(ctx, 0, "syslog_queue").Result(); if err != nil { time.Sleep(1 * time.Second); continue }
		var e LogEntry; if json.Unmarshal([]byte(res[1]), &e) != nil { continue }
		
		_, err = db.Exec(`INSERT INTO syslogs (timestamp, source_ip, protocol, hostname, app_name, severity, facility, facility_name, source_type, payload) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`, e.Timestamp, e.SourceIP, e.Protocol, e.Hostname, e.AppName, e.Severity, e.Facility, e.FacilityName, e.SourceType, e.Payload)
		if err != nil { rdb.LPush(ctx, "syslog_queue", res[1]); time.Sleep(2 * time.Second) }
	}
}

func retentionWorker() {
	for {
		var days int
		if db.QueryRow("SELECT retention_days FROM settings WHERE id = 1").Scan(&days) == nil && days > 0 { db.Exec(fmt.Sprintf("DELETE FROM syslogs WHERE timestamp < NOW() - INTERVAL '%d days'", days)) }
		time.Sleep(1 * time.Hour)
	}
}

// --- MÓDULO DE LOGS ---
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

func fetchLogsHTML(w http.ResponseWriter, r *http.Request) {
	query, args := buildLogsQuery(r, 50)
	rows, err := db.Query(query, args...)
	if err != nil { http.Error(w, "Erro", http.StatusInternalServerError); return }
	defer rows.Close()

	var logs []LogEntry
	for rows.Next() {
		var l LogEntry; var ts time.Time
		if rows.Scan(&l.ID, &ts, &l.SourceIP, &l.Protocol, &l.Hostname, &l.AppName, &l.Severity, &l.Facility, &l.FacilityName, &l.SourceType, &l.Payload) == nil {
			l.Timestamp = ts.Format("2006-01-02 15:04:05"); logs = append(logs, l)
		}
	}
	renderTemplate(w, "templates/logs_table.html", logs)
}

func exportCSV(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment;filename=syslogs.csv")
	w.Write([]byte{0xEF, 0xBB, 0xBF})
	writer := csv.NewWriter(w); defer writer.Flush()
	writer.Write([]string{"ID", "Data", "Origem IP", "Host", "App", "Source Type", "Facility", "Gravidade", "Msg"})
	query, args := buildLogsQuery(r, 2000); rows, _ := db.Query(query, args...); defer rows.Close()
	for rows.Next() { var l LogEntry; var ts time.Time; if rows.Scan(&l.ID, &ts, &l.SourceIP, &l.Protocol, &l.Hostname, &l.AppName, &l.Severity, &l.Facility, &l.FacilityName, &l.SourceType, &l.Payload) == nil { writer.Write([]string{fmt.Sprint(l.ID), ts.Format("2006-01-02 15:04:05"), l.SourceIP, l.Hostname, l.AppName, l.SourceType, l.FacilityName, l.Severity, l.Payload}) } }
}

// --- ESTATÍSTICAS ---
func serveStatsView(w http.ResponseWriter, r *http.Request) { renderTemplate(w, "templates/stats.html", nil) }

func fetchStatsData(w http.ResponseWriter, r *http.Request) {
	var res StatsResponse
	rows, _ := db.Query("SELECT severity, COUNT(*) FROM syslogs GROUP BY severity ORDER BY count DESC"); defer rows.Close()
	for rows.Next() { var s SeverityStat; rows.Scan(&s.Severity, &s.Count); res.Severities = append(res.Severities, s) }
	rows2, _ := db.Query("SELECT hostname, COUNT(*) FROM syslogs WHERE hostname != '-' GROUP BY hostname ORDER BY count DESC LIMIT 5"); defer rows2.Close()
	for rows2.Next() { var h HostStat; rows2.Scan(&h.Hostname, &h.Count); res.Hosts = append(res.Hosts, h) }
	rows3, _ := db.Query("SELECT source_type, COUNT(*) FROM syslogs GROUP BY source_type ORDER BY count DESC"); defer rows3.Close()
	for rows3.Next() { var src SourceStat; rows3.Scan(&src.Source, &src.Count); res.Sources = append(res.Sources, src) }
	w.Header().Set("Content-Type", "application/json"); json.NewEncoder(w).Encode(res)
}

// --- ALERTAS ---
func serveAlertsView(w http.ResponseWriter, r *http.Request) {
	var rules []AlertRule
	rows, _ := db.Query("SELECT id, enabled, name, severity, source_type, keyword, threshold, window_minutes FROM alert_rules ORDER BY id DESC")
	defer rows.Close()
	for rows.Next() { var ar AlertRule; rows.Scan(&ar.ID, &ar.Enabled, &ar.Name, &ar.Severity, &ar.SourceType, &ar.Keyword, &ar.Threshold, &ar.WindowMinutes); rules = append(rules, ar) }
	renderTemplate(w, "templates/alerts.html", rules)
}

func saveAlertRule(w http.ResponseWriter, r *http.Request) {
	db.Exec("INSERT INTO alert_rules (name, source_type, keyword, threshold, window_minutes) VALUES ($1, $2, $3, $4, $5)", r.FormValue("name"), r.FormValue("source_type"), r.FormValue("keyword"), r.FormValue("threshold"), r.FormValue("window"))
	w.Write([]byte(`<div class="p-3 bg-emerald-50 text-emerald-700 rounded-lg border border-emerald-200 text-sm flex items-center font-medium"><i data-lucide="check-circle" class="w-5 h-5 mr-2"></i> Regra base gravada. (Monitorização agendada para a próxima Sprint)</div><script>lucide.createIcons();</script>`))
}

// --- POLÍTICAS ---
func servePoliciesView(w http.ResponseWriter, r *http.Request) {
	var data PolicyViewData
	db.QueryRow("SELECT enabled, minimum_severity, ignored_apps, ignored_hosts, ignored_keywords FROM log_policies WHERE id=1").Scan(&data.Policy.Enabled, &data.Policy.MinimumSeverity, &data.AppsStr, &data.HostsStr, &data.KeywordsStr)
	data.Stats = map[string]string{ "TotalRecebidos": getRedisStat("stats:received_total"), "TotalGuardados": getRedisStat("stats:stored_total"), "TotalFiltrados": getRedisStat("stats:filtered_total"), "BySeverity": getRedisStat("stats:filtered_severity"), "ByApp": getRedisStat("stats:filtered_app"), "ByHost": getRedisStat("stats:filtered_host"), "ByKeyword": getRedisStat("stats:filtered_keyword") }
	renderTemplate(w, "templates/policies.html", data)
}

func getRedisStat(key string) string { val, err := rdb.Get(ctx, key).Result(); if err != nil || val == "" { return "0" }; return val }

func savePolicies(w http.ResponseWriter, r *http.Request) {
	enabled := r.FormValue("enabled") == "on"
	db.Exec("UPDATE log_policies SET enabled=$1, minimum_severity=$2, ignored_apps=$3, ignored_hosts=$4, ignored_keywords=$5 WHERE id=1", enabled, r.FormValue("min_severity"), r.FormValue("apps"), r.FormValue("hosts"), r.FormValue("keywords"))
	syncPolicyToRedis()
	w.Write([]byte(`<div class="p-3 bg-emerald-50 text-emerald-700 rounded-lg text-sm flex items-center font-medium"><i data-lucide="check-circle" class="w-5 h-5 mr-2"></i> Regras atualizadas no Collector com sucesso!</div><script>lucide.createIcons();</script>`))
}

// --- DEFINIÇÕES E FERRAMENTAS ---
func serveSettingsView(w http.ResponseWriter, r *http.Request) {
	var s Settings; db.QueryRow("SELECT retention_days, admin_user FROM settings WHERE id = 1").Scan(&s.Retention, &s.User)
	renderTemplate(w, "templates/settings.html", s)
}

func saveSettings(w http.ResponseWriter, r *http.Request) {
	retention, user, pass := r.FormValue("retention"), r.FormValue("username"), r.FormValue("password")
	if pass != "" { db.Exec("UPDATE settings SET retention_days = $1, admin_user = $2, admin_pass = $3 WHERE id = 1", retention, user, pass) } else { db.Exec("UPDATE settings SET retention_days = $1, admin_user = $2 WHERE id = 1", retention, user) }
	w.Write([]byte(`<div class="p-3 bg-emerald-50 text-emerald-700 rounded-lg text-sm flex items-center font-medium"><i data-lucide="check-circle" class="w-5 h-5 mr-2"></i> Atualizado com sucesso!</div><script>lucide.createIcons();</script>`))
}

func serveToolsView(w http.ResponseWriter, r *http.Request) { renderTemplate(w, "templates/tools.html", nil) }

func downloadTool(w http.ResponseWriter, r *http.Request) {
	var c, f string
	switch r.URL.Query().Get("action") {
	case "firewall": c = "@echo off\necho A abrir a Firewall...\nstart wf.msc\nexit"; f = "abrir_firewall.bat"
	case "compmgmt": c = "@echo off\necho A abrir Permissoes...\nstart compmgmt.msc\nexit"; f = "gerir_permissoes.bat"
	case "secpol": c = "@echo off\necho A abrir Secpol...\nstart secpol.msc\nexit"; f = "politicas_seguranca.bat"
	default: http.Error(w, "Inválido", 400); return
	}
	w.Header().Set("Content-Disposition", "attachment; filename="+f); w.Header().Set("Content-Type", "application/bat")
	w.Write([]byte(c))
}