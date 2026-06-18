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
	ID        int
	Timestamp string `json:"timestamp"`
	SourceIP  string `json:"source_ip"`
	Protocol  string `json:"protocol"`
	Hostname  string `json:"hostname"`
	AppName   string `json:"app_name"`
	Severity  string `json:"severity"`
	Facility  string `json:"facility"`
	Payload   string `json:"payload"`
}

type SeverityStat struct { Severity string `json:"severity"`; Count int `json:"count"` }
type HostStat struct { Hostname string `json:"hostname"`; Count int `json:"count"` }
type StatsResponse struct { Severities []SeverityStat `json:"severities"`; Hosts []HostStat `json:"hosts"` }
type Settings struct { Retention int; User string; Error string }

// NOVA ESTRUTURA: Políticas de Logs
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

	// Rotas Protegidas (HTML/HTMX)
	http.HandleFunc("/", authMiddleware(serveDashboard))
	http.HandleFunc("/logs", authMiddleware(fetchLogsHTML))
	http.HandleFunc("/export", authMiddleware(exportCSV))
	http.HandleFunc("/stats/view", authMiddleware(serveStatsView))
	http.HandleFunc("/api/stats", authMiddleware(fetchStatsData))
	http.HandleFunc("/settings/view", authMiddleware(serveSettingsView))
	http.HandleFunc("/settings/save", authMiddleware(saveSettings))
	http.HandleFunc("/tools/view", authMiddleware(serveToolsView))
	http.HandleFunc("/tools/download", authMiddleware(downloadTool))
	
	// NOVAS ROTAS DE POLÍTICAS
	http.HandleFunc("/policies/view", authMiddleware(servePoliciesView))
	http.HandleFunc("/policies/save", authMiddleware(savePolicies))

	log.Println("Painel de Administração ativo na porta 8080")
	http.ListenAndServe(":8080", nil)
}

// --- MIDDLEWARES ---
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		validPaths := map[string]bool{ "/": true, "/logs": true, "/export": true, "/stats/view": true, "/api/stats": true, "/settings/view": true, "/settings/save": true, "/tools/view": true, "/tools/download": true, "/policies/view": true, "/policies/save": true }
		if !validPaths[r.URL.Path] { http.NotFound(w, r); return }

		cookie, err := r.Cookie("admin_session")
		if err != nil || cookie.Value != "valid" {
			if r.Header.Get("HX-Request") == "true" { w.Header().Set("HX-Redirect", "/login"); w.WriteHeader(http.StatusUnauthorized) } else { http.Redirect(w, r, "/login", http.StatusSeeOther) }
			return
		}
		next.ServeHTTP(w, r)
	}
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

	tmpl := `<!DOCTYPE html><html lang="pt-PT"><head><meta charset="UTF-8"><title>Login - Log Center</title><script src="https://cdn.tailwindcss.com"></script><script>fetch('/styles.css').then(r=>r.text()).then(c=>{const s=document.createElement('style');s.type='text/tailwindcss';s.innerHTML=c;document.head.appendChild(s);});</script></head>
	<body class="bg-slate-900 flex items-center justify-center h-screen relative">
		<div class="absolute inset-0 overflow-hidden opacity-20 bg-[url('data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iNDAiIGhlaWdodD0iNDAiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyI+PGRlZnM+PHBhdHRlcm4gaWQ9ImciIHdpZHRoPSI0MCIgaGVpZ2h0PSI0MCIgcGF0dGVyblVuaXRzPSJ1c2VyU3BhY2VPblVzZSI+PHBhdGggZD0iTTAgNDBoNDBWMEgwem0yMCAyMGMxMS4wNDYgMCAyMC04Ljk1NCAyMC0yMFMyOC45NTQgMCAxOCAwIDAgOC45NTQgMCAyMHMyMC04Ljk1NCAyMC0yMHoiIGZpbGw9IiNmZmYiIGZpbGwtcnVsZT0iZXZlbm9kZCIvPjwvcGF0dGVybj48L2RlZnM+PHJlY3Qgd2lkdGg9IjEwMCUiIGhlaWdodD0iMTAwJSIgZmlsbD0idXJsKCNnKSIvPjwvc3ZnPg==')]"></div>
		<div class="card p-10 max-w-sm w-full z-10">
			<div class="flex justify-center mb-4"><div class="bg-blue-100 p-3 rounded-full"><svg class="w-8 h-8 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"></path></svg></div></div>
			<h2 class="text-2xl font-bold text-center text-slate-800 mb-2">Log Center</h2>
			<p class="text-center text-slate-500 text-sm mb-6">Autentique-se para aceder ao sistema</p>
			{{if .Error}}<div class="bg-red-50 text-red-600 text-sm p-3 rounded-lg text-center mb-4 border border-red-200">{{.Error}}</div>{{end}}
			<form method="POST" action="/login" class="space-y-5">
				<div><label class="form-label">Utilizador</label><input type="text" name="username" class="form-input" required autofocus></div>
				<div><label class="form-label">Palavra-passe</label><input type="password" name="password" class="form-input" required></div>
				<button type="submit" class="btn-primary w-full mt-2">Iniciar Sessão</button>
			</form>
		</div>
	</body></html>`
	t, _ := template.New("login").Parse(tmpl); t.Execute(w, s)
}

// --- INIT (DB & REDIS) ---
func initRedis() {
	rdb = redis.NewClient(&redis.Options{Addr: os.Getenv("REDIS_URL"), DB: 0})
	if _, err := rdb.Ping(ctx).Result(); err != nil { log.Fatalf("Erro Redis: %v", err) }
}

func initDB() {
	connStr := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable", os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASS"), os.Getenv("DB_NAME"))
	for i := 0; i < 5; i++ { db, _ = sql.Open("postgres", connStr); if db.Ping() == nil { break }; time.Sleep(3 * time.Second) }
	
	db.Exec(`CREATE TABLE IF NOT EXISTS syslogs (id SERIAL PRIMARY KEY, timestamp TIMESTAMP, source_ip VARCHAR(50), protocol VARCHAR(10), hostname VARCHAR(100) DEFAULT '-', app_name VARCHAR(100) DEFAULT '-', severity VARCHAR(20) DEFAULT 'Info', facility VARCHAR(100) DEFAULT '-', payload TEXT, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP); CREATE INDEX IF NOT EXISTS idx_timestamp ON syslogs(timestamp DESC);`)
	db.Exec(`CREATE TABLE IF NOT EXISTS settings (id SERIAL PRIMARY KEY, retention_days INT DEFAULT 30, admin_user VARCHAR(100) DEFAULT 'admin', admin_pass VARCHAR(100) DEFAULT 'admin');`)
	db.Exec(`INSERT INTO settings (id) SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM settings WHERE id = 1)`)

	// Criar Tabela de Políticas
	db.Exec(`CREATE TABLE IF NOT EXISTS log_policies (id SERIAL PRIMARY KEY, enabled BOOLEAN DEFAULT false, minimum_severity VARCHAR(20) DEFAULT 'Info', ignored_apps TEXT DEFAULT '', ignored_hosts TEXT DEFAULT '', ignored_keywords TEXT DEFAULT '');`)
	db.Exec(`INSERT INTO log_policies (id) SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM log_policies WHERE id = 1)`)

	// Enviar a política inicial do Postgres para o Redis
	syncPolicyToRedis()
}

// SINCRONIZADOR DE POLÍTICAS (PG -> Redis Pub/Sub)
func syncPolicyToRedis() {
	var enabled bool; var minSev, appsStr, hostsStr, kwStr string
	db.QueryRow("SELECT enabled, minimum_severity, ignored_apps, ignored_hosts, ignored_keywords FROM log_policies WHERE id=1").Scan(&enabled, &minSev, &appsStr, &hostsStr, &kwStr)

	policy := LogPolicy{
		Enabled:         enabled,
		MinimumSeverity: minSev,
		IgnoredApps:     parseList(appsStr),
		IgnoredHosts:    parseList(hostsStr),
		IgnoredKeywords: parseList(kwStr),
	}

	jsonData, _ := json.Marshal(policy)
	rdb.Set(ctx, "active_log_policy", jsonData, 0)
	rdb.Publish(ctx, "policy_updates", "reload") // Avisa o Collector para recarregar!
}

func parseList(input string) []string {
	parts := strings.Split(input, ",")
	var res []string
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" { res = append(res, t) }
	}
	return res
}

// --- WORKERS ---
func logWorker() {
	defer func() { if r := recover(); r != nil { time.Sleep(2 * time.Second); go logWorker() } }()
	for {
		result, err := rdb.BRPop(ctx, 0, "syslog_queue").Result()
		if err != nil { time.Sleep(1 * time.Second); continue }
		var entry LogEntry
		if json.Unmarshal([]byte(result[1]), &entry) != nil { continue }
		_, err = db.Exec(`INSERT INTO syslogs (timestamp, source_ip, protocol, hostname, app_name, severity, payload, facility) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`, entry.Timestamp, entry.SourceIP, entry.Protocol, entry.Hostname, entry.AppName, entry.Severity, entry.Payload, entry.Facility)
		if err != nil { rdb.LPush(ctx, "syslog_queue", result[1]); time.Sleep(2 * time.Second) }
	}
}

func retentionWorker() {
	for {
		var days int
		if db.QueryRow("SELECT retention_days FROM settings WHERE id = 1").Scan(&days) == nil && days > 0 {
			db.Exec(fmt.Sprintf("DELETE FROM syslogs WHERE timestamp < NOW() - INTERVAL '%d days'", days))
		}
		time.Sleep(1 * time.Hour)
	}
}

// --- MÓDULO DE LOGS ---
func buildLogsQuery(r *http.Request, limit int) (string, []interface{}) {
	q, sev := r.URL.Query().Get("q"), r.URL.Query().Get("sev")
	query := "SELECT id, timestamp, source_ip, protocol, hostname, app_name, severity, facility, payload FROM syslogs WHERE 1=1"
	var args []interface{}
	argId := 1

	if q != "" { query += fmt.Sprintf(" AND (source_ip ILIKE $%d OR payload ILIKE $%d OR hostname ILIKE $%d OR app_name ILIKE $%d OR facility ILIKE $%d)", argId, argId, argId, argId, argId); args = append(args, "%"+q+"%"); argId++ }
	if sev != "" { query += fmt.Sprintf(" AND severity = $%d", argId); args = append(args, sev); argId++ }
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
		if rows.Scan(&l.ID, &ts, &l.SourceIP, &l.Protocol, &l.Hostname, &l.AppName, &l.Severity, &l.Facility, &l.Payload) == nil {
			l.Timestamp = ts.Format("2006-01-02 15:04:05")
			logs = append(logs, l)
		}
	}

	tmpl := `
	{{range .}}
	<tr class="tr-row">
		<td class="td-cell font-medium text-slate-900">#{{.ID}}</td>
		<td class="td-cell text-slate-500 whitespace-nowrap">{{.Timestamp}}</td>
		<td class="td-cell"><div class="font-mono text-slate-700">{{.SourceIP}}</div><div class="text-xs text-slate-400 mt-0.5">{{.Protocol}}</div></td>
		<td class="td-cell"><div class="font-medium text-slate-800 truncate max-w-[150px]" title="{{.Hostname}}">{{.Hostname}}</div><div class="text-xs text-slate-500 truncate max-w-[150px]" title="{{.AppName}} ({{.Facility}})">{{.AppName}} <span class="text-slate-400">({{.Facility}})</span></div></td>
		<td class="td-cell">
			<span class="badge 
				{{if eq .Severity "Emergência"}} badge-emergencia
				{{else if eq .Severity "Alerta"}} badge-alerta
				{{else if eq .Severity "Crítico"}} badge-critico
				{{else if eq .Severity "Erro"}} badge-erro
				{{else if eq .Severity "Aviso"}} badge-aviso
				{{else if eq .Severity "Notice"}} badge-notice
				{{else if eq .Severity "Debug"}} badge-debug
				{{else}} badge-info {{end}}">
				{{.Severity}}
			</span>
		</td>
		<td class="td-cell font-mono text-xs text-slate-600">
			<div class="truncate max-w-[200px] xl:max-w-md">{{.Payload}}</div>
			<button onclick="openLogDetails(this)" data-id="{{.ID}}" data-ts="{{.Timestamp}}" data-ip="{{.SourceIP}}" data-proto="{{.Protocol}}" data-host="{{.Hostname}}" data-app="{{.AppName}}" data-sev="{{.Severity}}" data-fac="{{.Facility}}" data-payload="{{.Payload}}" class="mt-1.5 text-blue-600 hover:text-blue-800 hover:underline font-sans font-semibold text-[11px] flex items-center transition-colors">
				Ver detalhes &rarr;
			</button>
		</td>
	</tr>
	{{else}}
	<tr><td colspan="6" class="text-center py-12 text-slate-500 bg-slate-50"><svg class="mx-auto h-12 w-12 text-slate-400 mb-2" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 13h6m-3-3v6m-9 1V7a2 2 0 012-2h6l2 2h6a2 2 0 012 2v8a2 2 0 01-2 2H5a2 2 0 01-2-2z" /></svg><p class="text-sm font-medium">Nenhum registo encontrado.</p></td></tr>
	{{end}}`
	t, _ := template.New("logs").Parse(tmpl); t.Execute(w, logs)
}

func exportCSV(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment;filename=syslogs_cm_oliveira_hospital.csv")
	w.Write([]byte{0xEF, 0xBB, 0xBF})
	writer := csv.NewWriter(w); defer writer.Flush()
	writer.Write([]string{"ID", "Data/Hora", "Origem (IP)", "Protocolo", "Hostname", "App", "Gravidade", "Facility", "Mensagem"})
	query, args := buildLogsQuery(r, 2000)
	rows, _ := db.Query(query, args...); defer rows.Close()
	for rows.Next() {
		var l LogEntry; var ts time.Time
		if rows.Scan(&l.ID, &ts, &l.SourceIP, &l.Protocol, &l.Hostname, &l.AppName, &l.Severity, &l.Facility, &l.Payload) == nil {
			writer.Write([]string{fmt.Sprintf("%d", l.ID), ts.Format("2006-01-02 15:04:05"), l.SourceIP, l.Protocol, l.Hostname, l.AppName, l.Severity, l.Facility, l.Payload})
		}
	}
}

// --- POLÍTICAS DE LOGS ---
func servePoliciesView(w http.ResponseWriter, r *http.Request) {
	var data PolicyViewData
	db.QueryRow("SELECT enabled, minimum_severity, ignored_apps, ignored_hosts, ignored_keywords FROM log_policies WHERE id=1").Scan(
		&data.Policy.Enabled, &data.Policy.MinimumSeverity, &data.AppsStr, &data.HostsStr, &data.KeywordsStr,
	)

	data.Stats = map[string]string{
		"TotalRecebidos": getRedisStat("stats:received_total"),
		"TotalGuardados": getRedisStat("stats:stored_total"),
		"TotalFiltrados": getRedisStat("stats:filtered_total"),
		"BySeverity":     getRedisStat("stats:filtered_severity"),
		"ByApp":          getRedisStat("stats:filtered_app"),
		"ByHost":         getRedisStat("stats:filtered_host"),
		"ByKeyword":      getRedisStat("stats:filtered_keyword"),
	}

	tmpl := `
	<div class="space-y-6 fade-in">
		<div class="flex justify-between items-center mb-2">
			<h2 class="text-2xl font-bold text-slate-800">Motor de Políticas de Logs</h2>
			<p class="text-sm text-slate-500">Bloqueie lixo na origem para poupar disco e processamento.</p>
		</div>

		<div class="grid grid-cols-1 xl:grid-cols-3 gap-6">
			<div class="card p-6 xl:col-span-2">
				<form hx-post="/policies/save" hx-target="#policy-msg" class="space-y-5">
					<div class="flex items-center space-x-3 p-4 bg-slate-50 rounded-lg border border-slate-200">
						<input type="checkbox" name="enabled" id="enabled" class="w-5 h-5 text-blue-600 rounded border-gray-300 focus:ring-blue-500" {{if .Policy.Enabled}}checked{{end}}>
						<div>
							<label for="enabled" class="font-bold text-slate-700 cursor-pointer">Ativar Filtros e Políticas de Bloqueio</label>
							<p class="text-xs text-slate-500">Se desativado, o sistema aceitará todos os logs recebidos.</p>
						</div>
					</div>

					<div>
						<label class="form-label"><i data-lucide="filter" class="w-4 h-4 inline mr-1 text-slate-400"></i> Severidade Mínima</label>
						<select name="min_severity" class="form-input cursor-pointer">
							<option value="Debug" {{if eq .Policy.MinimumSeverity "Debug"}}selected{{end}}>Debug (Guardar Tudo)</option>
							<option value="Info" {{if eq .Policy.MinimumSeverity "Info"}}selected{{end}}>Info (Padrão)</option>
							<option value="Notice" {{if eq .Policy.MinimumSeverity "Notice"}}selected{{end}}>Notice</option>
							<option value="Aviso" {{if eq .Policy.MinimumSeverity "Aviso"}}selected{{end}}>Aviso (Apenas Avisos e Erros)</option>
							<option value="Erro" {{if eq .Policy.MinimumSeverity "Erro"}}selected{{end}}>Erro (Ignorar Avisos, focar em problemas)</option>
						</select>
					</div>

					<div>
						<label class="form-label"><i data-lucide="box" class="w-4 h-4 inline mr-1 text-slate-400"></i> Aplicações Ignoradas (Separadas por vírgula)</label>
						<input type="text" name="apps" value="{{.AppsStr}}" placeholder="ex: CRON, systemd, Microsoft-Windows-Kernel..." class="form-input">
					</div>

					<div>
						<label class="form-label"><i data-lucide="server" class="w-4 h-4 inline mr-1 text-slate-400"></i> Hosts/Servidores Ignorados</label>
						<input type="text" name="hosts" value="{{.HostsStr}}" placeholder="ex: PC-Recepcao, SW-Sala1..." class="form-input">
					</div>

					<div>
						<label class="form-label"><i data-lucide="type" class="w-4 h-4 inline mr-1 text-slate-400"></i> Palavras-chave Ignoradas</label>
						<textarea name="keywords" rows="3" placeholder="ex: favicon.ico, healthcheck..." class="form-input">{{.KeywordsStr}}</textarea>
					</div>

					<div class="pt-3">
						<button type="submit" class="btn-primary"><i data-lucide="save" class="w-4 h-4 mr-2"></i> Aplicar Políticas em Tempo Real</button>
					</div>
					<div id="policy-msg" class="mt-4"></div>
				</form>
			</div>

			<div class="card p-6 bg-slate-900 text-white flex flex-col justify-between">
				<div>
					<h3 class="text-lg font-bold flex items-center text-blue-400 mb-6"><i data-lucide="shield-alert" class="w-5 h-5 mr-2"></i> Tráfego Protegido</h3>
					<div class="space-y-4 font-mono text-sm">
						<div class="flex justify-between border-b border-slate-700 pb-2"><span class="text-slate-400">Recebidos (UDP/TCP):</span><span class="font-bold text-blue-400">{{.Stats.TotalRecebidos}}</span></div>
						<div class="flex justify-between border-b border-slate-700 pb-2"><span class="text-slate-400">Guardados na BD:</span><span class="font-bold text-emerald-400">{{.Stats.TotalGuardados}}</span></div>
						<div class="flex justify-between border-b border-slate-700 pb-2 pt-2"><span class="text-slate-300 font-bold uppercase tracking-wider">Bloqueados Totais:</span><span class="font-bold text-red-400">{{.Stats.TotalFiltrados}}</span></div>
						
						<div class="pt-4 space-y-2 text-xs text-slate-400 pl-4 border-l-2 border-slate-700">
							<div class="flex justify-between"><span>Motivo: Severidade</span><span>{{.Stats.BySeverity}}</span></div>
							<div class="flex justify-between"><span>Motivo: App Bloqueada</span><span>{{.Stats.ByApp}}</span></div>
							<div class="flex justify-between"><span>Motivo: Host Bloqueado</span><span>{{.Stats.ByHost}}</span></div>
							<div class="flex justify-between"><span>Motivo: Palavra-chave</span><span>{{.Stats.ByKeyword}}</span></div>
						</div>
					</div>
				</div>
				<button hx-get="/policies/view" hx-target="#main-content" class="mt-8 text-center text-xs text-slate-400 hover:text-white transition-colors bg-slate-800 py-2 rounded-lg border border-slate-700"><i data-lucide="refresh-cw" class="w-3 h-3 inline mr-1"></i> Atualizar Contadores</button>
			</div>
		</div>
	</div>
	<style>.fade-in { animation: fadeIn 0.3s ease-in-out; } @keyframes fadeIn { from { opacity: 0; transform: translateY(10px); } to { opacity: 1; transform: translateY(0); } }</style>
	<script>lucide.createIcons();</script>
	`
	t, _ := template.New("policies").Parse(tmpl); t.Execute(w, data)
}

func getRedisStat(key string) string {
	val, err := rdb.Get(ctx, key).Result()
	if err != nil || val == "" { return "0" }
	return val
}

func savePolicies(w http.ResponseWriter, r *http.Request) {
	enabled := r.FormValue("enabled") == "on"
	db.Exec("UPDATE log_policies SET enabled=$1, minimum_severity=$2, ignored_apps=$3, ignored_hosts=$4, ignored_keywords=$5 WHERE id=1",
		enabled, r.FormValue("min_severity"), r.FormValue("apps"), r.FormValue("hosts"), r.FormValue("keywords"))
	
	syncPolicyToRedis() // Sincroniza em tempo real com o Collector

	w.Write([]byte(`<div class="p-3 bg-emerald-50 text-emerald-700 rounded-lg border border-emerald-200 text-sm flex items-center font-medium"><i data-lucide="check-circle" class="w-5 h-5 mr-2"></i> Regras atualizadas e injetadas no Collector com sucesso!</div><script>lucide.createIcons();</script>`))
}

// --- OUTRAS VISTAS (Estatísticas, Ferramentas, Settings) ---
func serveStatsView(w http.ResponseWriter, r *http.Request) {
	tmpl := `<div class="space-y-6 fade-in"><div class="flex justify-between items-center mb-2"><h2 class="text-2xl font-bold text-slate-800">Análise e Estatísticas</h2></div><div class="grid grid-cols-1 md:grid-cols-2 gap-6"><div class="card p-6"><h3 class="text-lg font-semibold text-slate-700 mb-4 flex items-center"><i data-lucide="pie-chart" class="w-5 h-5 mr-2 text-blue-500"></i> Distribuição por Gravidade</h3><div class="relative h-72 w-full flex justify-center"><canvas id="severityChart"></canvas></div></div><div class="card p-6"><h3 class="text-lg font-semibold text-slate-700 mb-4 flex items-center"><i data-lucide="server" class="w-5 h-5 mr-2 text-emerald-500"></i> Top 5 Hosts Mais Ativos</h3><div class="relative h-72 w-full"><canvas id="hostsChart"></canvas></div></div></div></div><style>.fade-in { animation: fadeIn 0.3s ease-in-out; } @keyframes fadeIn { from { opacity: 0; transform: translateY(10px); } to { opacity: 1; transform: translateY(0); } }</style><script>lucide.createIcons(); fetch('/api/stats').then(res => res.json()).then(data => { const colorMap = { 'Emergência': '#b91c1c', 'Alerta': '#ea580c', 'Crítico': '#ef4444', 'Erro': '#f87171', 'Aviso': '#eab308', 'Notice': '#3b82f6', 'Info': '#10b981', 'Debug': '#64748b' }; const bgColors = data.severities.map(s => colorMap[s.severity] || '#cbd5e1'); new Chart(document.getElementById('severityChart').getContext('2d'), { type: 'doughnut', data: { labels: data.severities.map(s => s.severity), datasets: [{ data: data.severities.map(s => s.count), backgroundColor: bgColors, borderWidth: 0, hoverOffset: 4 }] }, options: { maintainAspectRatio: false, plugins: { legend: { position: 'right', labels: { usePointStyle: true, boxWidth: 8 } } }, cutout: '65%' } }); new Chart(document.getElementById('hostsChart').getContext('2d'), { type: 'bar', data: { labels: data.hosts.map(h => h.hostname), datasets: [{ label: 'Nº de Logs Registados', data: data.hosts.map(h => h.count), backgroundColor: '#3b82f6', borderRadius: 6, barThickness: 32 }] }, options: { maintainAspectRatio: false, plugins: { legend: { display: false } }, scales: { y: { beginAtZero: true, grid: { borderDash: [4, 4] } }, x: { grid: { display: false } } } } }); });</script>`
	w.Header().Set("Content-Type", "text/html"); w.Write([]byte(tmpl))
}

func fetchStatsData(w http.ResponseWriter, r *http.Request) {
	var res StatsResponse
	rows, _ := db.Query("SELECT severity, COUNT(*) FROM syslogs GROUP BY severity ORDER BY count DESC")
	defer rows.Close(); for rows.Next() { var s SeverityStat; rows.Scan(&s.Severity, &s.Count); res.Severities = append(res.Severities, s) }
	rows2, _ := db.Query("SELECT hostname, COUNT(*) FROM syslogs WHERE hostname != '-' GROUP BY hostname ORDER BY count DESC LIMIT 5")
	defer rows2.Close(); for rows2.Next() { var h HostStat; rows2.Scan(&h.Hostname, &h.Count); res.Hosts = append(res.Hosts, h) }
	w.Header().Set("Content-Type", "application/json"); json.NewEncoder(w).Encode(res)
}

func serveSettingsView(w http.ResponseWriter, r *http.Request) {
	var s Settings
	db.QueryRow("SELECT retention_days, admin_user FROM settings WHERE id = 1").Scan(&s.Retention, &s.User)
	tmpl := `<div class="space-y-6 fade-in"><div class="flex justify-between items-center mb-2"><h2 class="text-2xl font-bold text-slate-800">Definições do Sistema</h2></div><div class="card p-6 max-w-2xl"><form hx-post="/settings/save" hx-target="#settings-msg" class="space-y-5"><div><label class="form-label"><i data-lucide="clock" class="w-4 h-4 inline mr-1 text-slate-400"></i> Retenção de Logs (Dias)</label><input type="number" name="retention" value="{{.Retention}}" min="1" class="form-input"></div><hr class="border-slate-100 my-4"><div><label class="form-label"><i data-lucide="user" class="w-4 h-4 inline mr-1 text-slate-400"></i> Administrador</label><input type="text" name="username" value="{{.User}}" required class="form-input"></div><div><label class="form-label"><i data-lucide="key" class="w-4 h-4 inline mr-1 text-slate-400"></i> Nova Palavra-passe</label><input type="password" name="password" placeholder="Deixe vazio para manter a atual" class="form-input"></div><div class="pt-3"><button type="submit" class="btn-primary"><i data-lucide="save" class="w-4 h-4 mr-2"></i> Guardar</button></div><div id="settings-msg" class="mt-4"></div></form></div></div><style>.fade-in { animation: fadeIn 0.3s ease-in-out; } @keyframes fadeIn { from { opacity: 0; transform: translateY(10px); } to { opacity: 1; transform: translateY(0); } }</style><script>lucide.createIcons();</script>`
	t, _ := template.New("settings").Parse(tmpl); t.Execute(w, s)
}

func saveSettings(w http.ResponseWriter, r *http.Request) {
	retention, user, pass := r.FormValue("retention"), r.FormValue("username"), r.FormValue("password")
	if pass != "" { db.Exec("UPDATE settings SET retention_days = $1, admin_user = $2, admin_pass = $3 WHERE id = 1", retention, user, pass) } else { db.Exec("UPDATE settings SET retention_days = $1, admin_user = $2 WHERE id = 1", retention, user) }
	w.Write([]byte(`<div class="p-3 bg-emerald-50 text-emerald-700 rounded-lg border border-emerald-200 text-sm flex items-center font-medium"><i data-lucide="check-circle" class="w-5 h-5 mr-2"></i> Atualizado com sucesso!</div><script>lucide.createIcons();</script>`))
}

func serveToolsView(w http.ResponseWriter, r *http.Request) {
	tmpl := `
	<div class="space-y-6 fade-in">
		<div class="flex justify-between items-center mb-2">
			<h2 class="text-2xl font-bold text-slate-800">Ferramentas de Sistema</h2>
		</div>
		<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
			<div class="tool-card hover:border-blue-300">
				<div class="tool-icon-box bg-blue-50"><i data-lucide="network" class="text-blue-600"></i></div>
				<h3 class="text-md font-bold text-slate-800 mb-1">Definições de Rede</h3>
				<p class="text-xs text-slate-500 mb-4">Abre o painel nativo de rede do Windows.</p>
				<a href="ms-settings:network" class="inline-flex items-center text-sm font-semibold text-blue-600 hover:underline">Abrir Painel <i data-lucide="external-link" class="w-4 h-4 ml-1"></i></a>
			</div>
			<div class="tool-card hover:border-emerald-300">
				<div class="tool-icon-box bg-emerald-50"><i data-lucide="shield-alert" class="text-emerald-600"></i></div>
				<h3 class="text-md font-bold text-slate-800 mb-1">Segurança Windows</h3>
				<p class="text-xs text-slate-500 mb-4">Verifique proteção contra ameaças.</p>
				<a href="windowsdefender://" class="inline-flex items-center text-sm font-semibold text-emerald-600 hover:underline">Abrir Segurança <i data-lucide="external-link" class="w-4 h-4 ml-1"></i></a>
			</div>
			<div class="tool-card hover:border-red-300">
				<div class="tool-icon-box bg-red-50"><i data-lucide="flame" class="text-red-600"></i></div>
				<h3 class="text-md font-bold text-slate-800 mb-1">Firewall Avançada</h3>
				<p class="text-xs text-slate-500 mb-4">Gestor de regras de entrada/saída (wf.msc).</p>
				<a href="/tools/download?action=firewall" class="inline-flex items-center text-sm font-semibold text-red-600 hover:underline bg-red-50 px-3 py-1.5 rounded border border-red-100">Gatilho (.bat) <i data-lucide="download" class="w-4 h-4 ml-1"></i></a>
			</div>
			<div class="tool-card hover:border-purple-300">
				<div class="tool-icon-box bg-purple-50"><i data-lucide="users" class="text-purple-600"></i></div>
				<h3 class="text-md font-bold text-slate-800 mb-1">Permissões</h3>
				<p class="text-xs text-slate-500 mb-4">Gerir utilizadores e grupos locais.</p>
				<a href="/tools/download?action=compmgmt" class="inline-flex items-center text-sm font-semibold text-purple-600 hover:underline bg-purple-50 px-3 py-1.5 rounded border border-purple-100">Gatilho (.bat) <i data-lucide="download" class="w-4 h-4 ml-1"></i></a>
			</div>
			<div class="tool-card hover:border-orange-300">
				<div class="tool-icon-box bg-orange-50"><i data-lucide="lock" class="text-orange-600"></i></div>
				<h3 class="text-md font-bold text-slate-800 mb-1">Políticas Locais</h3>
				<p class="text-xs text-slate-500 mb-4">Acesso às políticas de auditoria.</p>
				<a href="/tools/download?action=secpol" class="inline-flex items-center text-sm font-semibold text-orange-600 hover:underline bg-orange-50 px-3 py-1.5 rounded border border-orange-100">Gatilho (.bat) <i data-lucide="download" class="w-4 h-4 ml-1"></i></a>
			</div>
		</div>
	</div>
	<style>.fade-in { animation: fadeIn 0.3s ease-in-out; } @keyframes fadeIn { from { opacity: 0; transform: translateY(10px); } to { opacity: 1; transform: translateY(0); } }</style>
	<script>lucide.createIcons();</script>
	`
	w.Header().Set("Content-Type", "text/html"); w.Write([]byte(tmpl))
}

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