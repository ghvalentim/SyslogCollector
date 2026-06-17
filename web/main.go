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
	"time"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

// Estrutura do Log
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

// Estruturas para Gráficos
type SeverityStat struct {
	Severity string `json:"severity"`
	Count    int    `json:"count"`
}

type HostStat struct {
	Hostname string `json:"hostname"`
	Count    int    `json:"count"`
}

type StatsResponse struct {
	Severities []SeverityStat `json:"severities"`
	Hosts      []HostStat     `json:"hosts"`
}

// Estrutura para as Definições e Login
type Settings struct {
	Retention int
	User      string
	Error     string
}

var (
	db  *sql.DB
	rdb *redis.Client
	ctx = context.Background()
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Fatalf("[CRÍTICO] Falha fatal no serviço Web/Worker: %v", r)
		}
	}()

	log.Println("A iniciar Syslog Web Panel e Worker da base de dados...")

	initRedis()
	initDB()

	// Worker 1: Processa os logs do Redis e guarda no Postgres
	go logWorker()
	// Worker 2: Limpeza automática de logs antigos
	go retentionWorker()

	// Rotas Públicas (Login)
	http.HandleFunc("/login", serveLogin)
	http.HandleFunc("/logout", handleLogout)

	// Rotas Protegidas (Envolvidas no Middleware de Autenticação)
	http.HandleFunc("/", authMiddleware(serveDashboard))
	http.HandleFunc("/logs", authMiddleware(fetchLogsHTML))
	http.HandleFunc("/export", authMiddleware(exportCSV))
	http.HandleFunc("/stats/view", authMiddleware(serveStatsView))
	http.HandleFunc("/api/stats", authMiddleware(fetchStatsData))
	http.HandleFunc("/settings/view", authMiddleware(serveSettingsView))
	http.HandleFunc("/settings/save", authMiddleware(saveSettings))

	log.Println("Painel de Administração ativo na porta 8080 (Interno)")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Erro ao iniciar servidor HTTP: %v", err)
	}
}

// --- MIDDLEWARE DE AUTENTICAÇÃO ---
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// A rota raiz ("/") no mux pode apanhar lixo, garantimos que só serve o index exato
		if r.URL.Path != "/" && r.URL.Path != "/logs" && r.URL.Path != "/export" && r.URL.Path != "/stats/view" && r.URL.Path != "/api/stats" && r.URL.Path != "/settings/view" && r.URL.Path != "/settings/save" {
			http.NotFound(w, r)
			return
		}

		cookie, err := r.Cookie("admin_session")
		if err != nil || cookie.Value != "valid" {
			// Se for um pedido HTMX, envia cabeçalho de redirecionamento especial
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

// --- LÓGICA DE LOGIN ---
func serveLogin(w http.ResponseWriter, r *http.Request) {
	var s Settings

	if r.Method == "POST" {
		user := r.FormValue("username")
		pass := r.FormValue("password")
		var dbUser, dbPass string

		db.QueryRow("SELECT admin_user, admin_pass FROM settings WHERE id = 1").Scan(&dbUser, &dbPass)

		if user == dbUser && pass == dbPass {
			http.SetCookie(w, &http.Cookie{
				Name:     "admin_session",
				Value:    "valid",
				Path:     "/",
				HttpOnly: true,
				MaxAge:   86400, // Sessão de 1 dia
			})
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		s.Error = "Credenciais incorretas!"
	}

	tmpl := `<!DOCTYPE html>
	<html lang="pt-PT">
	<head><meta charset="UTF-8"><title>Login - Log Center</title><script src="https://cdn.tailwindcss.com"></script></head>
	<body class="bg-slate-900 flex items-center justify-center h-screen relative">
		<div class="absolute inset-0 overflow-hidden opacity-20 bg-[url('data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iNDAiIGhlaWdodD0iNDAiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyI+PGRlZnM+PHBhdHRlcm4gaWQ9ImciIHdpZHRoPSI0MCIgaGVpZ2h0PSI0MCIgcGF0dGVyblVuaXRzPSJ1c2VyU3BhY2VPblVzZSI+PHBhdGggZD0iTTAgNDBoNDBWMEgwem0yMCAyMGMxMS4wNDYgMCAyMC04Ljk1NCAyMC0yMFMyOC45NTQgMCAxOCAwIDAgOC45NTQgMCAyMHMyMC04Ljk1NCAyMC0yMHoiIGZpbGw9IiNmZmYiIGZpbGwtcnVsZT0iZXZlbm9kZCIvPjwvcGF0dGVybj48L2RlZnM+PHJlY3Qgd2lkdGg9IjEwMCUiIGhlaWdodD0iMTAwJSIgZmlsbD0idXJsKCNnKSIvPjwvc3ZnPg==')]"></div>
		<div class="bg-white p-10 rounded-2xl shadow-2xl w-full max-w-sm z-10">
			<div class="flex justify-center mb-4"><div class="bg-blue-100 p-3 rounded-full"><svg class="w-8 h-8 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"></path></svg></div></div>
			<h2 class="text-2xl font-bold text-center text-slate-800 mb-2">Log Center</h2>
			<p class="text-center text-slate-500 text-sm mb-6">Autentique-se para aceder ao sistema</p>
			{{if .Error}}<div class="bg-red-50 text-red-600 text-sm p-3 rounded-lg text-center mb-4 border border-red-200">{{.Error}}</div>{{end}}
			<form method="POST" action="/login" class="space-y-5">
				<div><label class="block text-xs font-bold text-slate-500 uppercase tracking-wider mb-1">Utilizador</label><input type="text" name="username" class="w-full px-4 py-2.5 bg-slate-50 border border-slate-200 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 text-sm" required autofocus></div>
				<div><label class="block text-xs font-bold text-slate-500 uppercase tracking-wider mb-1">Palavra-passe</label><input type="password" name="password" class="w-full px-4 py-2.5 bg-slate-50 border border-slate-200 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 text-sm" required></div>
				<button type="submit" class="w-full bg-blue-600 text-white font-semibold py-2.5 px-4 rounded-lg hover:bg-blue-700 transition-colors">Iniciar Sessão</button>
			</form>
		</div>
	</body>
	</html>`
	t, _ := template.New("login").Parse(tmpl)
	t.Execute(w, s)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "admin_session", Value: "", Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func initRedis() {
	redisURL := os.Getenv("REDIS_URL")
	rdb = redis.NewClient(&redis.Options{Addr: redisURL, DB: 0})
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		log.Fatalf("[ERRO] Falha ao conectar ao Redis: %v", err)
	}
}

func initDB() {
	connStr := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASS"), os.Getenv("DB_NAME"))

	var err error
	for i := 0; i < 5; i++ {
		db, err = sql.Open("postgres", connStr)
		if err == nil {
			err = db.Ping()
			if err == nil {
				break
			}
		}
		log.Println("A aguardar pela base de dados...")
		time.Sleep(3 * time.Second)
	}

	if err != nil {
		log.Fatalf("[ERRO] Impossível conectar ao PostgreSQL: %v", err)
	}

	// Criar Tabela de Logs
	query := `
	CREATE TABLE IF NOT EXISTS syslogs (
		id SERIAL PRIMARY KEY,
		timestamp TIMESTAMP,
		source_ip VARCHAR(50),
		protocol VARCHAR(10),
		payload TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_timestamp ON syslogs(timestamp DESC);
	`
	db.Exec(query)

	// Migrações de Logs
	db.Exec("ALTER TABLE syslogs ADD COLUMN hostname VARCHAR(100) DEFAULT '-'")
	db.Exec("ALTER TABLE syslogs ADD COLUMN app_name VARCHAR(100) DEFAULT '-'")
	db.Exec("ALTER TABLE syslogs ADD COLUMN severity VARCHAR(20) DEFAULT 'Info'")
	db.Exec("ALTER TABLE syslogs ADD COLUMN facility VARCHAR(100) DEFAULT '-'")

	// Criar Tabela de Definições (Settings)
	querySettings := `
	CREATE TABLE IF NOT EXISTS settings (
		id SERIAL PRIMARY KEY,
		retention_days INT DEFAULT 30,
		admin_user VARCHAR(100) DEFAULT 'admin',
		admin_pass VARCHAR(100) DEFAULT 'admin'
	);
	`
	db.Exec(querySettings)

	// Injetar credenciais padrão se a tabela estiver vazia
	db.Exec(`INSERT INTO settings (id, retention_days, admin_user, admin_pass) 
	         SELECT 1, 30, 'admin', 'admin' 
	         WHERE NOT EXISTS (SELECT 1 FROM settings WHERE id = 1)`)

	log.Println("Base de dados sincronizada com sucesso.")
}

func logWorker() {
	defer func() {
		if r := recover(); r != nil {
			time.Sleep(2 * time.Second)
			go logWorker()
		}
	}()

	for {
		result, err := rdb.BRPop(ctx, 0, "syslog_queue").Result()
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		jsonData := result[1]
		var entry LogEntry
		if err := json.Unmarshal([]byte(jsonData), &entry); err != nil {
			continue
		}

		query := `INSERT INTO syslogs (timestamp, source_ip, protocol, hostname, app_name, severity, payload, facility) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
		_, err = db.Exec(query, entry.Timestamp, entry.SourceIP, entry.Protocol, entry.Hostname, entry.AppName, entry.Severity, entry.Payload, entry.Facility)
		if err != nil {
			rdb.LPush(ctx, "syslog_queue", jsonData)
			time.Sleep(2 * time.Second)
		}
	}
}

// --- WORKER DE LIMPEZA AUTOMÁTICA (RETENÇÃO) ---
func retentionWorker() {
	for {
		var days int
		err := db.QueryRow("SELECT retention_days FROM settings WHERE id = 1").Scan(&days)
		if err == nil && days > 0 {
			// Apaga logs mais antigos do que X dias definidos
			query := fmt.Sprintf("DELETE FROM syslogs WHERE timestamp < NOW() - INTERVAL '%d days'", days)
			res, _ := db.Exec(query)
			if rowsAffected, err := res.RowsAffected(); err == nil && rowsAffected > 0 {
				log.Printf("[LIMPEZA] %d logs antigos foram apagados automaticamente (Retenção: %d dias).", rowsAffected, days)
			}
		}
		// Corre de hora a hora
		time.Sleep(1 * time.Hour)
	}
}

func serveDashboard(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "index.html")
}

func fetchLogsHTML(w http.ResponseWriter, r *http.Request) {
	queryParam := r.URL.Query().Get("q")
	
	var rows *sql.Rows
	var err error

	if queryParam != "" {
		searchTerm := "%" + queryParam + "%"
		rows, err = db.Query(`
			SELECT id, timestamp, source_ip, protocol, hostname, app_name, severity, facility, payload 
			FROM syslogs 
			WHERE source_ip ILIKE $1 OR payload ILIKE $1 OR hostname ILIKE $1 OR app_name ILIKE $1 OR facility ILIKE $1
			ORDER BY timestamp DESC LIMIT 50`, searchTerm)
	} else {
		rows, err = db.Query("SELECT id, timestamp, source_ip, protocol, hostname, app_name, severity, facility, payload FROM syslogs ORDER BY timestamp DESC LIMIT 50")
	}

	if err != nil {
		http.Error(w, "Erro ao procurar logs", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var logs []LogEntry
	for rows.Next() {
		var l LogEntry
		var ts time.Time
		if err := rows.Scan(&l.ID, &ts, &l.SourceIP, &l.Protocol, &l.Hostname, &l.AppName, &l.Severity, &l.Facility, &l.Payload); err == nil {
			l.Timestamp = ts.Format("2006-01-02 15:04:05")
			logs = append(logs, l)
		}
	}

	tmpl := `
	{{range .}}
	<tr class="border-b hover:bg-slate-50 transition-colors">
		<td class="px-4 py-3 text-sm font-medium text-slate-900">#{{.ID}}</td>
		<td class="px-4 py-3 text-sm text-slate-500 whitespace-nowrap">{{.Timestamp}}</td>
		<td class="px-4 py-3 text-sm">
			<div class="font-mono text-slate-700">{{.SourceIP}}</div>
			<div class="text-xs text-slate-400 mt-0.5">{{.Protocol}}</div>
		</td>
		<td class="px-4 py-3 text-sm">
			<div class="font-medium text-slate-800 truncate max-w-[150px]" title="{{.Hostname}}">{{.Hostname}}</div>
			<div class="text-xs text-slate-500 truncate max-w-[150px]" title="{{.AppName}} ({{.Facility}})">{{.AppName}} <span class="text-slate-400">({{.Facility}})</span></div>
		</td>
		<td class="px-4 py-3 text-sm">
			<span class="px-2 py-1 text-[10px] uppercase tracking-wider font-bold rounded-md border 
				{{if eq .Severity "Emergência"}} bg-red-100 text-red-800 border-red-200
				{{else if eq .Severity "Alerta"}} bg-orange-100 text-orange-800 border-orange-200
				{{else if eq .Severity "Crítico"}} bg-red-50 text-red-700 border-red-200
				{{else if eq .Severity "Erro"}} bg-red-50 text-red-600 border-red-100
				{{else if eq .Severity "Aviso"}} bg-yellow-50 text-yellow-700 border-yellow-200
				{{else if eq .Severity "Notice"}} bg-blue-50 text-blue-700 border-blue-200
				{{else if eq .Severity "Debug"}} bg-slate-100 text-slate-600 border-slate-200
				{{else}} bg-emerald-50 text-emerald-700 border-emerald-200 {{end}}">
				{{.Severity}}
			</span>
		</td>
		<td class="px-4 py-3 text-sm text-slate-600 font-mono text-xs">
			<div class="truncate max-w-[200px] xl:max-w-md">{{.Payload}}</div>
			<button onclick="openLogDetails(this)" 
				data-id="{{.ID}}" data-ts="{{.Timestamp}}" data-ip="{{.SourceIP}}" data-proto="{{.Protocol}}" 
				data-host="{{.Hostname}}" data-app="{{.AppName}}" data-sev="{{.Severity}}" data-fac="{{.Facility}}" data-payload="{{.Payload}}" 
				class="mt-1.5 text-blue-600 hover:text-blue-800 hover:underline font-sans font-semibold text-[11px] flex items-center transition-colors">
				Ver detalhes &rarr;
			</button>
		</td>
	</tr>
	{{else}}
	<tr><td colspan="6" class="text-center py-12 text-slate-500 bg-slate-50"><p class="text-sm font-medium">Nenhum registo encontrado.</p></td></tr>
	{{end}}
	`
	t, _ := template.New("logs").Parse(tmpl)
	t.Execute(w, logs)
}

func exportCSV(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment;filename=syslogs_cm_oliveira_hospital.csv")
	w.Write([]byte{0xEF, 0xBB, 0xBF})

	writer := csv.NewWriter(w)
	defer writer.Flush()

	writer.Write([]string{"ID", "Data/Hora", "Origem (IP)", "Protocolo", "Hostname", "App", "Gravidade", "Facility", "Mensagem"})
	rows, _ := db.Query("SELECT id, timestamp, source_ip, protocol, hostname, app_name, severity, facility, payload FROM syslogs ORDER BY timestamp DESC LIMIT 2000")
	defer rows.Close()

	for rows.Next() {
		var l LogEntry
		var ts time.Time
		if err := rows.Scan(&l.ID, &ts, &l.SourceIP, &l.Protocol, &l.Hostname, &l.AppName, &l.Severity, &l.Facility, &l.Payload); err == nil {
			writer.Write([]string{fmt.Sprintf("%d", l.ID), ts.Format("2006-01-02 15:04:05"), l.SourceIP, l.Protocol, l.Hostname, l.AppName, l.Severity, l.Facility, l.Payload})
		}
	}
}

// --- ROTAS DAS DEFINIÇÕES ---
func serveSettingsView(w http.ResponseWriter, r *http.Request) {
	var s Settings
	db.QueryRow("SELECT retention_days, admin_user FROM settings WHERE id = 1").Scan(&s.Retention, &s.User)

	tmpl := `
	<div class="space-y-6 fade-in">
		<div class="flex justify-between items-center mb-2">
			<h2 class="text-2xl font-bold text-slate-800">Definições do Sistema</h2>
			<p class="text-sm text-slate-500">Gira a retenção de dados e as credenciais de acesso.</p>
		</div>
		<div class="bg-white p-6 rounded-xl shadow-sm border border-slate-200 max-w-2xl">
			<form hx-post="/settings/save" hx-target="#settings-msg" class="space-y-5">
				<div>
					<label class="block text-sm font-bold text-slate-700 mb-1"><i data-lucide="clock" class="w-4 h-4 inline mr-1 text-slate-400"></i> Retenção de Logs (Dias)</label>
					<input type="number" name="retention" value="{{.Retention}}" min="1" class="w-full px-4 py-2.5 bg-slate-50 border border-slate-200 rounded-lg focus:outline-none focus:border-blue-500 text-sm">
					<p class="text-xs text-slate-500 mt-1.5">Logs mais antigos que este valor serão permanentemente apagados da base de dados de hora a hora para poupar espaço.</p>
				</div>
				<hr class="border-slate-100 my-4">
				<div>
					<label class="block text-sm font-bold text-slate-700 mb-1"><i data-lucide="user" class="w-4 h-4 inline mr-1 text-slate-400"></i> Utilizador de Acesso (Administrador)</label>
					<input type="text" name="username" value="{{.User}}" required class="w-full px-4 py-2.5 bg-slate-50 border border-slate-200 rounded-lg focus:outline-none focus:border-blue-500 text-sm">
				</div>
				<div>
					<label class="block text-sm font-bold text-slate-700 mb-1"><i data-lucide="key" class="w-4 h-4 inline mr-1 text-slate-400"></i> Nova Palavra-passe</label>
					<input type="password" name="password" placeholder="Deixe em branco para não alterar a palavra-passe atual..." class="w-full px-4 py-2.5 bg-slate-50 border border-slate-200 rounded-lg focus:outline-none focus:border-blue-500 text-sm">
				</div>
				<div class="pt-3">
					<button type="submit" class="bg-blue-600 text-white font-semibold px-6 py-2.5 rounded-lg hover:bg-blue-700 transition-colors flex items-center text-sm shadow-sm">
						<i data-lucide="save" class="w-4 h-4 mr-2"></i> Guardar Alterações
					</button>
				</div>
				<div id="settings-msg" class="mt-4"></div>
			</form>
		</div>
	</div>
	<style>
		.fade-in { animation: fadeIn 0.3s ease-in-out; }
		@keyframes fadeIn { from { opacity: 0; transform: translateY(10px); } to { opacity: 1; transform: translateY(0); } }
	</style>
	<script>lucide.createIcons();</script>
	`
	t, _ := template.New("settings").Parse(tmpl)
	t.Execute(w, s)
}

func saveSettings(w http.ResponseWriter, r *http.Request) {
	retention := r.FormValue("retention")
	user := r.FormValue("username")
	pass := r.FormValue("password")

	if pass != "" {
		db.Exec("UPDATE settings SET retention_days = $1, admin_user = $2, admin_pass = $3 WHERE id = 1", retention, user, pass)
	} else {
		db.Exec("UPDATE settings SET retention_days = $1, admin_user = $2 WHERE id = 1", retention, user)
	}

	w.Write([]byte(`<div class="p-3 bg-emerald-50 text-emerald-700 rounded-lg border border-emerald-200 text-sm flex items-center font-medium"><i data-lucide="check-circle" class="w-5 h-5 mr-2"></i> Definições atualizadas com sucesso! As alterações já estão ativas.</div><script>lucide.createIcons();</script>`))
}

func serveStatsView(w http.ResponseWriter, r *http.Request) {
	// ... (Código mantido idêntico) ...
	tmpl := `
	<div class="space-y-6 fade-in">
		<div class="flex justify-between items-center mb-2">
			<h2 class="text-2xl font-bold text-slate-800">Análise e Estatísticas</h2>
		</div>
		<div class="grid grid-cols-1 md:grid-cols-2 gap-6">
			<div class="bg-white p-6 rounded-xl shadow-sm border border-slate-200"><h3 class="text-lg font-semibold text-slate-700 mb-4 flex items-center"><i data-lucide="pie-chart" class="w-5 h-5 mr-2 text-blue-500"></i> Distribuição por Gravidade</h3><div class="relative h-72 w-full flex justify-center"><canvas id="severityChart"></canvas></div></div>
			<div class="bg-white p-6 rounded-xl shadow-sm border border-slate-200"><h3 class="text-lg font-semibold text-slate-700 mb-4 flex items-center"><i data-lucide="server" class="w-5 h-5 mr-2 text-emerald-500"></i> Top 5 Hosts Mais Ativos</h3><div class="relative h-72 w-full"><canvas id="hostsChart"></canvas></div></div>
		</div>
	</div>
	<style>.fade-in { animation: fadeIn 0.3s ease-in-out; } @keyframes fadeIn { from { opacity: 0; transform: translateY(10px); } to { opacity: 1; transform: translateY(0); } }</style>
	<script>
		lucide.createIcons();
		fetch('/api/stats').then(res => res.json()).then(data => {
				const colorMap = { 'Emergência': '#b91c1c', 'Alerta': '#ea580c', 'Crítico': '#ef4444', 'Erro': '#f87171', 'Aviso': '#eab308', 'Notice': '#3b82f6', 'Info': '#10b981', 'Debug': '#64748b' };
				const bgColors = data.severities.map(s => colorMap[s.severity] || '#cbd5e1');
				new Chart(document.getElementById('severityChart').getContext('2d'), { type: 'doughnut', data: { labels: data.severities.map(s => s.severity), datasets: [{ data: data.severities.map(s => s.count), backgroundColor: bgColors, borderWidth: 0, hoverOffset: 4 }] }, options: { maintainAspectRatio: false, plugins: { legend: { position: 'right', labels: { usePointStyle: true, boxWidth: 8 } } }, cutout: '65%' } });
				new Chart(document.getElementById('hostsChart').getContext('2d'), { type: 'bar', data: { labels: data.hosts.map(h => h.hostname), datasets: [{ label: 'Nº de Logs Registados', data: data.hosts.map(h => h.count), backgroundColor: '#3b82f6', borderRadius: 6, barThickness: 32 }] }, options: { maintainAspectRatio: false, plugins: { legend: { display: false } }, scales: { y: { beginAtZero: true, grid: { borderDash: [4, 4] } }, x: { grid: { display: false } } } } });
		});
	</script>`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(tmpl))
}

func fetchStatsData(w http.ResponseWriter, r *http.Request) {
	var res StatsResponse
	rows, _ := db.Query("SELECT severity, COUNT(*) FROM syslogs GROUP BY severity ORDER BY count DESC")
	defer rows.Close()
	for rows.Next() { var s SeverityStat; rows.Scan(&s.Severity, &s.Count); res.Severities = append(res.Severities, s) }

	rows2, _ := db.Query("SELECT hostname, COUNT(*) FROM syslogs WHERE hostname != '-' GROUP BY hostname ORDER BY count DESC LIMIT 5")
	defer rows2.Close()
	for rows2.Next() { var h HostStat; rows2.Scan(&h.Hostname, &h.Count); res.Hosts = append(res.Hosts, h) }

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}