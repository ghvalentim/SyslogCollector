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

	http.HandleFunc("/login", serveLogin)
	http.HandleFunc("/logout", handleLogout)
	http.HandleFunc("/script.js", serveScript)
	http.HandleFunc("/output.css", serveStyles)

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
	
	// ROTAS ALERTAS
	http.HandleFunc("/alerts/view", authMiddleware(serveAlertsView))
	http.HandleFunc("/alerts/save", authMiddleware(saveAlertRule))

	log.Println("Painel de Administração ativo na porta 8080")
	http.ListenAndServe(":8080", nil)
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

func serveDashboard(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, "index.html") }
func serveScript(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, "script.js") }
func serveStyles(w http.ResponseWriter, r *http.Request) { w.Header().Set("Content-Type", "text/css"); http.ServeFile(w, r, "output.css") }
func handleLogout(w http.ResponseWriter, r *http.Request) { http.SetCookie(w, &http.Cookie{Name: "admin_session", Value: "", Path: "/", MaxAge: -1}); http.Redirect(w, r, "/login", http.StatusSeeOther) }

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

	tmpl := `<!DOCTYPE html><html lang="pt-PT"><head><meta charset="UTF-8"><title>Login - Log Center</title><link rel="stylesheet" href="/output.css"></head><body class="bg-slate-900 flex items-center justify-center h-screen relative"><div class="card p-10 max-w-sm w-full z-10"><h2 class="text-2xl font-bold text-center mb-6">Acesso Reservado</h2>{{if .Error}}<div class="bg-red-50 text-red-600 text-sm p-3 rounded-lg text-center mb-4 border border-red-200">{{.Error}}</div>{{end}}<form method="POST" action="/login" class="space-y-4"><div><label class="form-label">Utilizador</label><input type="text" name="username" class="form-input" required></div><div><label class="form-label">Palavra-passe</label><input type="password" name="password" class="form-input" required></div><button type="submit" class="btn-primary w-full mt-2 py-3">Entrar</button></form></div></body></html>`
	t, _ := template.New("login").Parse(tmpl); t.Execute(w, s)
}

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

	tmpl := `
	{{range .}}
	<tr class="tr-row">
		<td class="td-cell font-medium text-slate-900">#{{.ID}}</td>
		<td class="td-cell text-slate-500 whitespace-nowrap">{{.Timestamp}}</td>
		<td class="td-cell"><div class="font-mono text-slate-700">{{.SourceIP}}</div><div class="text-xs text-slate-400 mt-0.5">{{.Protocol}}</div></td>
		<td class="td-cell"><div class="font-medium text-slate-800 truncate max-w-[150px]" title="{{.Hostname}}">{{.Hostname}}</div><div class="text-xs text-slate-500 truncate max-w-[150px]" title="{{.AppName}}">{{.AppName}}</div></td>
		<td class="td-cell"><span class="px-2 py-1 bg-indigo-50 text-indigo-700 text-[10px] font-bold uppercase rounded border border-indigo-200">{{.SourceType}}</span></td>
		<td class="td-cell"><span class="badge {{if eq .Severity "Emergência"}}badge-emergencia{{else if eq .Severity "Alerta"}}badge-alerta{{else if eq .Severity "Crítico"}}badge-critico{{else if eq .Severity "Erro"}}badge-erro{{else if eq .Severity "Aviso"}}badge-aviso{{else if eq .Severity "Notice"}}badge-notice{{else if eq .Severity "Debug"}}badge-debug{{else}}badge-info{{end}}">{{.Severity}}</span></td>
		<td class="td-cell font-mono text-xs text-slate-600"><div class="truncate max-w-[200px] xl:max-w-md">{{.Payload}}</div>
			<button onclick="openLogDetails(this)" data-id="{{.ID}}" data-ts="{{.Timestamp}}" data-ip="{{.SourceIP}}" data-proto="{{.Protocol}}" data-host="{{.Hostname}}" data-app="{{.AppName}}" data-sev="{{.Severity}}" data-fac="{{.Facility}}" data-facname="{{.FacilityName}}" data-source="{{.SourceType}}" data-payload="{{.Payload}}" class="mt-1.5 text-blue-600 hover:text-blue-800 hover:underline font-sans font-semibold text-[11px] flex items-center transition-colors">Ver detalhes &rarr;</button>
		</td>
	</tr>
	{{else}}<tr><td colspan="7" class="text-center py-12 text-slate-500 bg-slate-50"><p class="text-sm font-medium">Nenhum registo encontrado.</p></td></tr>{{end}}`
	t, _ := template.New("logs").Parse(tmpl); t.Execute(w, logs)
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

func serveStatsView(w http.ResponseWriter, r *http.Request) {
	tmpl := `<div class="space-y-6 fade-in"><div class="flex justify-between items-center mb-2"><h2 class="text-2xl font-bold text-slate-800">Análise e Estatísticas</h2></div>
	<div class="grid grid-cols-1 md:grid-cols-3 gap-6">
		<div class="card p-6"><h3 class="text-lg font-semibold text-slate-700 mb-4 flex items-center"><i data-lucide="pie-chart" class="w-5 h-5 mr-2 text-blue-500"></i> Por Gravidade</h3><div class="relative h-64 w-full"><canvas id="severityChart"></canvas></div></div>
		<div class="card p-6"><h3 class="text-lg font-semibold text-slate-700 mb-4 flex items-center"><i data-lucide="network" class="w-5 h-5 mr-2 text-indigo-500"></i> Por Origem (Classificação)</h3><div class="relative h-64 w-full"><canvas id="sourceChart"></canvas></div></div>
		<div class="card p-6"><h3 class="text-lg font-semibold text-slate-700 mb-4 flex items-center"><i data-lucide="server" class="w-5 h-5 mr-2 text-emerald-500"></i> Top Hosts Ativos</h3><div class="relative h-64 w-full"><canvas id="hostsChart"></canvas></div></div>
	</div></div><style>.fade-in { animation: fadeIn 0.3s ease-in-out; }</style>
	<script>lucide.createIcons(); fetch('/api/stats').then(res => res.json()).then(data => { 
		new Chart(document.getElementById('severityChart').getContext('2d'), { type: 'doughnut', data: { labels: data.severities.map(s => s.severity), datasets: [{ data: data.severities.map(s => s.count), backgroundColor: data.severities.map(s => ({'Emergência':'#b91c1c','Alerta':'#ea580c','Crítico':'#ef4444','Erro':'#f87171','Aviso':'#eab308','Notice':'#3b82f6','Info':'#10b981','Debug':'#64748b'}[s.severity] || '#cbd5e1')), borderWidth: 0 }] }, options: { maintainAspectRatio: false, cutout: '65%', plugins:{legend:{position:'bottom'}} } }); 
		new Chart(document.getElementById('sourceChart').getContext('2d'), { type: 'doughnut', data: { labels: data.sources.map(s => s.source), datasets: [{ data: data.sources.map(s => s.count), backgroundColor: ['#6366f1', '#8b5cf6', '#a855f7', '#d946ef', '#ec4899', '#f43f5e', '#64748b'], borderWidth: 0 }] }, options: { maintainAspectRatio: false, cutout: '65%', plugins:{legend:{position:'bottom'}} } });
		new Chart(document.getElementById('hostsChart').getContext('2d'), { type: 'bar', data: { labels: data.hosts.map(h => h.hostname), datasets: [{ data: data.hosts.map(h => h.count), backgroundColor: '#3b82f6', borderRadius: 6 }] }, options: { maintainAspectRatio: false, plugins: { legend: { display: false } }, scales: { y: { beginAtZero: true, grid: { borderDash: [4, 4] } }, x: { grid: { display: false } } } } }); 
	});</script>`
	w.Header().Set("Content-Type", "text/html"); w.Write([]byte(tmpl))
}

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

func serveAlertsView(w http.ResponseWriter, r *http.Request) {
	var rules []AlertRule
	rows, _ := db.Query("SELECT id, enabled, name, severity, source_type, keyword, threshold, window_minutes FROM alert_rules ORDER BY id DESC")
	defer rows.Close()
	for rows.Next() { var ar AlertRule; rows.Scan(&ar.ID, &ar.Enabled, &ar.Name, &ar.Severity, &ar.SourceType, &ar.Keyword, &ar.Threshold, &ar.WindowMinutes); rules = append(rules, ar) }

	tmpl := `<div class="space-y-6 fade-in"><div class="flex justify-between items-center mb-2"><h2 class="text-2xl font-bold text-slate-800">Motor de Alertas</h2><p class="text-sm text-slate-500">Fundação de monitorização para notificação proativa (Mini-SIEM).</p></div>
	<div class="card p-6"><h3 class="text-lg font-bold mb-4">Nova Regra de Deteção</h3><form hx-post="/alerts/save" hx-target="#alert-msg" class="grid grid-cols-1 md:grid-cols-4 gap-4 items-end">
		<div class="md:col-span-2"><label class="form-label">Nome da Regra</label><input type="text" name="name" placeholder="ex: Disparo de Firewall" required class="form-input"></div>
		<div><label class="form-label">Origem</label><select name="source_type" class="form-input"><option value="">Qualquer</option><option value="Firewall">Firewall</option><option value="Windows">Windows</option></select></div>
		<div><label class="form-label">Palavra-Chave</label><input type="text" name="keyword" placeholder="ex: failed" class="form-input"></div>
		<div><label class="form-label">Ocorrências (X)</label><input type="number" name="threshold" value="5" class="form-input" required></div>
		<div><label class="form-label">Minutos (Y)</label><input type="number" name="window" value="5" class="form-input" required></div>
		<div class="md:col-span-2"><button type="submit" class="btn-primary w-full"><i data-lucide="plus" class="w-4 h-4 mr-2"></i> Criar Regra</button></div>
	</form><div id="alert-msg" class="mt-4"></div></div>
	<div class="card overflow-hidden"><table class="w-full text-left text-sm"><thead class="table-head"><tr><th class="th-cell">Estado</th><th class="th-cell">Regra</th><th class="th-cell">Condição</th><th class="th-cell">Threshold</th></tr></thead><tbody class="divide-y divide-slate-100">
	{{range .}}<tr><td class="td-cell"><span class="badge {{if .Enabled}}badge-info{{else}}badge-debug{{end}}">{{if .Enabled}}Ativa{{else}}Inativa{{end}}</span></td><td class="td-cell font-bold">{{.Name}}</td><td class="td-cell text-slate-500">Origem: {{if .SourceType}}{{.SourceType}}{{else}}Qualquer{{end}} | Chave: {{if .Keyword}}{{.Keyword}}{{else}}Qualquer{{end}}</td><td class="td-cell font-mono">{{.Threshold}} logs / {{.WindowMinutes}}m</td></tr>{{else}}<tr><td colspan="4" class="td-cell text-center text-slate-400">Sem regras definidas.</td></tr>{{end}}
	</tbody></table></div></div><style>.fade-in { animation: fadeIn 0.3s ease-in-out; }</style><script>lucide.createIcons();</script>`
	t, _ := template.New("alerts").Parse(tmpl); t.Execute(w, rules)
}

func saveAlertRule(w http.ResponseWriter, r *http.Request) {
	db.Exec("INSERT INTO alert_rules (name, source_type, keyword, threshold, window_minutes) VALUES ($1, $2, $3, $4, $5)", r.FormValue("name"), r.FormValue("source_type"), r.FormValue("keyword"), r.FormValue("threshold"), r.FormValue("window"))
	w.Write([]byte(`<div class="p-3 bg-emerald-50 text-emerald-700 rounded-lg border border-emerald-200 text-sm flex items-center font-medium"><i data-lucide="check-circle" class="w-5 h-5 mr-2"></i> Regra base gravada.</div><script>lucide.createIcons();</script>`))
}

func servePoliciesView(w http.ResponseWriter, r *http.Request) {
	var data PolicyViewData
	db.QueryRow("SELECT enabled, minimum_severity, ignored_apps, ignored_hosts, ignored_keywords FROM log_policies WHERE id=1").Scan(&data.Policy.Enabled, &data.Policy.MinimumSeverity, &data.AppsStr, &data.HostsStr, &data.KeywordsStr)
	data.Stats = map[string]string{ "TotalRecebidos": getRedisStat("stats:received_total"), "TotalGuardados": getRedisStat("stats:stored_total"), "TotalFiltrados": getRedisStat("stats:filtered_total"), "BySeverity": getRedisStat("stats:filtered_severity"), "ByApp": getRedisStat("stats:filtered_app"), "ByHost": getRedisStat("stats:filtered_host"), "ByKeyword": getRedisStat("stats:filtered_keyword") }

	tmpl := `<div class="space-y-6 fade-in"><div class="flex justify-between items-center mb-2"><h2 class="text-2xl font-bold text-slate-800">Motor de Políticas de Logs</h2><p class="text-sm text-slate-500">Bloqueie lixo na origem para poupar disco e processamento.</p></div>
		<div class="grid grid-cols-1 xl:grid-cols-3 gap-6"><div class="card p-6 xl:col-span-2"><form hx-post="/policies/save" hx-target="#policy-msg" class="space-y-5"><div class="flex items-center space-x-3 p-4 bg-slate-50 rounded-lg border border-slate-200"><input type="checkbox" name="enabled" id="enabled" class="w-5 h-5 text-blue-600 rounded border-gray-300" {{if .Policy.Enabled}}checked{{end}}><div><label for="enabled" class="font-bold text-slate-700 cursor-pointer">Ativar Filtros de Bloqueio</label></div></div><div><label class="form-label"><i data-lucide="filter" class="w-4 h-4 inline mr-1 text-slate-400"></i> Severidade Mínima</label><select name="min_severity" class="form-input cursor-pointer"><option value="Debug" {{if eq .Policy.MinimumSeverity "Debug"}}selected{{end}}>Debug</option><option value="Info" {{if eq .Policy.MinimumSeverity "Info"}}selected{{end}}>Info</option><option value="Notice" {{if eq .Policy.MinimumSeverity "Notice"}}selected{{end}}>Notice</option><option value="Aviso" {{if eq .Policy.MinimumSeverity "Aviso"}}selected{{end}}>Aviso</option><option value="Erro" {{if eq .Policy.MinimumSeverity "Erro"}}selected{{end}}>Erro</option></select></div><div><label class="form-label">Aplicações Ignoradas</label><input type="text" name="apps" value="{{.AppsStr}}" class="form-input"></div><div><label class="form-label">Hosts Ignorados</label><input type="text" name="hosts" value="{{.HostsStr}}" class="form-input"></div><div><label class="form-label">Palavras-chave Ignoradas</label><textarea name="keywords" rows="3" class="form-input">{{.KeywordsStr}}</textarea></div><div class="pt-3"><button type="submit" class="btn-primary"><i data-lucide="save" class="w-4 h-4 mr-2"></i> Aplicar Políticas em Tempo Real</button></div><div id="policy-msg" class="mt-4"></div></form></div>
		<div class="card p-6 bg-slate-900 text-white flex flex-col justify-between"><div><h3 class="text-lg font-bold flex items-center text-blue-400 mb-6"><i data-lucide="shield-alert" class="w-5 h-5 mr-2"></i> Tráfego Protegido</h3><div class="space-y-4 font-mono text-sm"><div class="flex justify-between border-b border-slate-700 pb-2"><span class="text-slate-400">Recebidos:</span><span class="font-bold text-blue-400">{{.Stats.TotalRecebidos}}</span></div><div class="flex justify-between border-b border-slate-700 pb-2"><span class="text-slate-400">Guardados:</span><span class="font-bold text-emerald-400">{{.Stats.TotalGuardados}}</span></div><div class="flex justify-between border-b border-slate-700 pb-2 pt-2"><span class="text-slate-300 font-bold">Bloqueados Totais:</span><span class="font-bold text-red-400">{{.Stats.TotalFiltrados}}</span></div><div class="pt-4 space-y-2 text-xs text-slate-400 pl-4 border-l-2 border-slate-700"><div class="flex justify-between"><span>Motivo: Severidade</span><span>{{.Stats.BySeverity}}</span></div><div class="flex justify-between"><span>Motivo: App Bloqueada</span><span>{{.Stats.ByApp}}</span></div><div class="flex justify-between"><span>Motivo: Host Bloqueado</span><span>{{.Stats.ByHost}}</span></div><div class="flex justify-between"><span>Motivo: Palavra-chave</span><span>{{.Stats.ByKeyword}}</span></div></div></div></div><button hx-get="/policies/view" hx-target="#main-content" class="mt-8 text-center text-xs text-slate-400 hover:text-white transition-colors bg-slate-800 py-2 rounded-lg border border-slate-700">Atualizar Contadores</button></div></div></div><style>.fade-in { animation: fadeIn 0.3s ease-in-out; }</style><script>lucide.createIcons();</script>`
	t, _ := template.New("policies").Parse(tmpl); t.Execute(w, data)
}

func getRedisStat(key string) string { val, err := rdb.Get(ctx, key).Result(); if err != nil || val == "" { return "0" }; return val }
func savePolicies(w http.ResponseWriter, r *http.Request) { enabled := r.FormValue("enabled") == "on"; db.Exec("UPDATE log_policies SET enabled=$1, minimum_severity=$2, ignored_apps=$3, ignored_hosts=$4, ignored_keywords=$5 WHERE id=1", enabled, r.FormValue("min_severity"), r.FormValue("apps"), r.FormValue("hosts"), r.FormValue("keywords")); syncPolicyToRedis(); w.Write([]byte(`<div class="p-3 bg-emerald-50 text-emerald-700 rounded-lg text-sm flex items-center font-medium"><i data-lucide="check-circle" class="w-5 h-5 mr-2"></i> Regras atualizadas e injetadas no Collector com sucesso!</div><script>lucide.createIcons();</script>`)) }

func serveSettingsView(w http.ResponseWriter, r *http.Request) {
	var s Settings; db.QueryRow("SELECT retention_days, admin_user FROM settings WHERE id = 1").Scan(&s.Retention, &s.User)
	tmpl := `<div class="space-y-6 fade-in"><div class="flex justify-between items-center mb-2"><h2 class="text-2xl font-bold text-slate-800">Definições do Sistema</h2></div><div class="card p-6 max-w-2xl"><form hx-post="/settings/save" hx-target="#settings-msg" class="space-y-5"><div><label class="form-label"><i data-lucide="clock" class="w-4 h-4 inline mr-1 text-slate-400"></i> Retenção de Logs (Dias)</label><input type="number" name="retention" value="{{.Retention}}" min="1" class="form-input"></div><hr class="border-slate-100 my-4"><div><label class="form-label"><i data-lucide="user" class="w-4 h-4 inline mr-1 text-slate-400"></i> Administrador</label><input type="text" name="username" value="{{.User}}" required class="form-input"></div><div><label class="form-label"><i data-lucide="key" class="w-4 h-4 inline mr-1 text-slate-400"></i> Nova Palavra-passe</label><input type="password" name="password" placeholder="Deixe vazio para manter a atual" class="form-input"></div><div class="pt-3"><button type="submit" class="btn-primary"><i data-lucide="save" class="w-4 h-4 mr-2"></i> Guardar</button></div><div id="settings-msg" class="mt-4"></div></form></div></div><style>.fade-in { animation: fadeIn 0.3s ease-in-out; }</style><script>lucide.createIcons();</script>`
	t, _ := template.New("settings").Parse(tmpl); t.Execute(w, s)
}

func saveSettings(w http.ResponseWriter, r *http.Request) { retention, user, pass := r.FormValue("retention"), r.FormValue("username"), r.FormValue("password"); if pass != "" { db.Exec("UPDATE settings SET retention_days = $1, admin_user = $2, admin_pass = $3 WHERE id = 1", retention, user, pass) } else { db.Exec("UPDATE settings SET retention_days = $1, admin_user = $2 WHERE id = 1", retention, user) }; w.Write([]byte(`<div class="p-3 bg-emerald-50 text-emerald-700 rounded-lg text-sm flex items-center font-medium"><i data-lucide="check-circle" class="w-5 h-5 mr-2"></i> Atualizado com sucesso!</div><script>lucide.createIcons();</script>`)) }
func serveToolsView(w http.ResponseWriter, r *http.Request) { tmpl := `<div class="space-y-6 fade-in"><div class="flex justify-between items-center mb-2"><h2 class="text-2xl font-bold text-slate-800">Ferramentas de Sistema</h2></div><div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6"><div class="tool-card hover:border-blue-300"><div class="tool-icon-box bg-blue-50"><i data-lucide="network" class="text-blue-600"></i></div><h3 class="text-md font-bold text-slate-800 mb-1">Definições de Rede</h3><p class="text-xs text-slate-500 mb-4">Abre o painel nativo de rede do Windows.</p><a href="ms-settings:network" class="inline-flex items-center text-sm font-semibold text-blue-600 hover:underline">Abrir Painel</a></div><div class="tool-card hover:border-red-300"><div class="tool-icon-box bg-red-50"><i data-lucide="flame" class="text-red-600"></i></div><h3 class="text-md font-bold text-slate-800 mb-1">Firewall Avançada</h3><p class="text-xs text-slate-500 mb-4">Gestor de regras de entrada/saída (wf.msc).</p><a href="/tools/download?action=firewall" class="inline-flex items-center text-sm font-semibold text-red-600 hover:underline">Gatilho (.bat) <i data-lucide="download" class="w-4 h-4 ml-1"></i></a></div></div></div><style>.fade-in { animation: fadeIn 0.3s ease-in-out; }</style><script>lucide.createIcons();</script>`; w.Header().Set("Content-Type", "text/html"); w.Write([]byte(tmpl)) }
func downloadTool(w http.ResponseWriter, r *http.Request) { var c, f string; switch r.URL.Query().Get("action") { case "firewall": c = "@echo off\necho A abrir a Firewall...\nstart wf.msc\nexit"; f = "abrir_firewall.bat"; default: http.Error(w, "Inválido", 400); return }; w.Header().Set("Content-Disposition", "attachment; filename="+f); w.Header().Set("Content-Type", "application/bat"); w.Write([]byte(c)) }