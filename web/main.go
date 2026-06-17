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

// Estrutura do Log com o campo facility integrado
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

// Estruturas para os Gráficos de Estatísticas
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

	// Worker que processa os logs do Redis e guarda no Postgres
	go logWorker()

	// Rotas HTTP
	http.HandleFunc("/", serveDashboard)
	http.HandleFunc("/logs", fetchLogsHTML)
	http.HandleFunc("/export", exportCSV)
	
	// Novas Rotas para Estatísticas
	http.HandleFunc("/stats/view", serveStatsView)
	http.HandleFunc("/api/stats", fetchStatsData)

	log.Println("Painel de Administração ativo na porta 8080 (Interno)")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Erro ao iniciar servidor HTTP: %v", err)
	}
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
	_, err = db.Exec(query)
	if err != nil {
		log.Fatalf("[ERRO] Falha ao criar tabela: %v", err)
	}

	// Migrações seguras: Adiciona as novas colunas se não existirem
	db.Exec("ALTER TABLE syslogs ADD COLUMN hostname VARCHAR(100) DEFAULT '-'")
	db.Exec("ALTER TABLE syslogs ADD COLUMN app_name VARCHAR(100) DEFAULT '-'")
	db.Exec("ALTER TABLE syslogs ADD COLUMN severity VARCHAR(20) DEFAULT 'Info'")
	db.Exec("ALTER TABLE syslogs ADD COLUMN facility VARCHAR(100) DEFAULT '-'")

	log.Println("Base de dados sincronizada com sucesso.")
}

func logWorker() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[ERRO] Panic no Worker de processamento: %v\n", r)
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
			// Devolve à fila em caso de erro na BD
			rdb.LPush(ctx, "syslog_queue", jsonData)
			time.Sleep(2 * time.Second)
		}
	}
}

func serveDashboard(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "index.html")
}

func fetchLogsHTML(w http.ResponseWriter, r *http.Request) {
	queryParam := r.URL.Query().Get("q")
	
	var rows *sql.Rows
	var err error

	// Corrigido: adicionada a coluna 'facility' no SELECT das duas queries abaixo
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
		// Agora o Scan possui 9 colunas exatamente mapeadas a partir da Query
		if err := rows.Scan(&l.ID, &ts, &l.SourceIP, &l.Protocol, &l.Hostname, &l.AppName, &l.Severity, &l.Facility, &l.Payload); err == nil {
			l.Timestamp = ts.Format("2006-01-02 15:04:05")
			logs = append(logs, l)
		} else {
			log.Printf("[ERRO] Falha ao ler linha do PostgreSQL: %v\n", err)
		}
	}

	// Adicionámos a Facility no ecrã como um subtítulo elegante ao lado da aplicação
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
				data-id="{{.ID}}" 
				data-ts="{{.Timestamp}}" 
				data-ip="{{.SourceIP}}" 
				data-proto="{{.Protocol}}" 
				data-host="{{.Hostname}}" 
				data-app="{{.AppName}}" 
				data-sev="{{.Severity}}" 
				data-fac="{{.Facility}}" 
				data-payload="{{.Payload}}" 
				class="mt-1.5 text-blue-600 hover:text-blue-800 hover:underline font-sans font-semibold text-[11px] flex items-center transition-colors">
				Ver detalhes &rarr;
			</button>
		</td>
	</tr>
	{{else}}
	<tr>
		<td colspan="6" class="text-center py-12 text-slate-500 bg-slate-50">
			<svg class="mx-auto h-12 w-12 text-slate-400 mb-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 13h6m-3-3v6m-9 1V7a2 2 0 012-2h6l2 2h6a2 2 0 012 2v8a2 2 0 01-2 2H5a2 2 0 01-2-2z" />
			</svg>
			<p class="text-sm font-medium">Nenhum registo encontrado.</p>
		</td>
	</tr>
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

	queryParam := r.URL.Query().Get("q")
	var rows *sql.Rows
	var err error

	if queryParam != "" {
		searchTerm := "%" + queryParam + "%"
		rows, err = db.Query("SELECT id, timestamp, source_ip, protocol, hostname, app_name, severity, facility, payload FROM syslogs WHERE source_ip ILIKE $1 OR payload ILIKE $1 OR hostname ILIKE $1 OR app_name ILIKE $1 OR facility ILIKE $1 ORDER BY timestamp DESC LIMIT 2000", searchTerm)
	} else {
		rows, err = db.Query("SELECT id, timestamp, source_ip, protocol, hostname, app_name, severity, facility, payload FROM syslogs ORDER BY timestamp DESC LIMIT 2000")
	}

	if err != nil {
		http.Error(w, "Erro ao exportar logs", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var l LogEntry
		var ts time.Time
		if err := rows.Scan(&l.ID, &ts, &l.SourceIP, &l.Protocol, &l.Hostname, &l.AppName, &l.Severity, &l.Facility, &l.Payload); err == nil {
			writer.Write([]string{
				fmt.Sprintf("%d", l.ID),
				ts.Format("2006-01-02 15:04:05"),
				l.SourceIP,
				l.Protocol,
				l.Hostname,
				l.AppName,
				l.Severity,
				l.Facility,
				l.Payload,
			})
		}
	}
}

// Servir o HTML do Dashboard de Estatísticas via HTMX
func serveStatsView(w http.ResponseWriter, r *http.Request) {
	tmpl := `
	<div class="space-y-6 fade-in">
		<div class="flex justify-between items-center mb-2">
			<h2 class="text-2xl font-bold text-slate-800">Análise e Estatísticas</h2>
			<p class="text-sm text-slate-500">Dados baseados em todos os registos processados do sistema</p>
		</div>
		<div class="grid grid-cols-1 md:grid-cols-2 gap-6">
			<!-- Gráfico Circular -->
			<div class="bg-white p-6 rounded-xl shadow-sm border border-slate-200">
				<h3 class="text-lg font-semibold text-slate-700 mb-4 flex items-center"><i data-lucide="pie-chart" class="w-5 h-5 mr-2 text-blue-500"></i> Distribuição por Gravidade</h3>
				<div class="relative h-72 w-full flex justify-center">
					<canvas id="severityChart"></canvas>
				</div>
			</div>
			<!-- Gráfico de Barras -->
			<div class="bg-white p-6 rounded-xl shadow-sm border border-slate-200">
				<h3 class="text-lg font-semibold text-slate-700 mb-4 flex items-center"><i data-lucide="server" class="w-5 h-5 mr-2 text-emerald-500"></i> Top 5 Hosts Mais Ativos</h3>
				<div class="relative h-72 w-full">
					<canvas id="hostsChart"></canvas>
				</div>
			</div>
		</div>
	</div>
	<style>
		.fade-in { animation: fadeIn 0.3s ease-in-out; }
		@keyframes fadeIn { from { opacity: 0; transform: translateY(10px); } to { opacity: 1; transform: translateY(0); } }
	</style>
	<script>
		lucide.createIcons(); // Recarregar ícones para o novo DOM injetado
		
		// Ir buscar o JSON com os dados consolidados e renderizar gráficos
		fetch('/api/stats')
			.then(response => response.json())
			.then(data => {
				
				// Dicionário Fixo de Cores: "Emergência" será sempre vermelho escuro!
				const colorMap = {
					'Emergência': '#b91c1c', // Vermelho escuro
					'Alerta': '#ea580c',     // Laranja escuro
					'Crítico': '#ef4444',    // Vermelho padrão
					'Erro': '#f87171',       // Vermelho claro
					'Aviso': '#eab308',      // Amarelo
					'Notice': '#3b82f6',     // Azul
					'Info': '#10b981',       // Verde Esmeralda
					'Debug': '#64748b'       // Cinzento
				};

				// Mapeia a cor baseada no nome gravidade do log, usando cinzento como fallback
				const bgColors = data.severities.map(s => colorMap[s.severity] || '#cbd5e1');

				// Configurar Gráfico de Gravidade (Doughnut)
				const sevCtx = document.getElementById('severityChart').getContext('2d');
				new Chart(sevCtx, {
					type: 'doughnut',
					data: {
						labels: data.severities.map(s => s.severity),
						datasets: [{
							data: data.severities.map(s => s.count),
							backgroundColor: bgColors,
							borderWidth: 0,
							hoverOffset: 4
						}]
					},
					options: { 
						maintainAspectRatio: false, 
						plugins: { 
							legend: { position: 'right', labels: { usePointStyle: true, boxWidth: 8 } } 
						},
						cutout: '65%'
					}
				});

				// Configurar Gráfico de Hosts (Barras)
				const hostCtx = document.getElementById('hostsChart').getContext('2d');
				new Chart(hostCtx, {
					type: 'bar',
					data: {
						labels: data.hosts.map(h => h.hostname),
						datasets: [{
							label: 'Nº de Logs Registados',
							data: data.hosts.map(h => h.count),
							backgroundColor: '#3b82f6',
							borderRadius: 6,
							barThickness: 32
						}]
					},
					options: { 
						maintainAspectRatio: false,
						plugins: { legend: { display: false } },
						scales: { 
							y: { beginAtZero: true, grid: { borderDash: [4, 4] } },
							x: { grid: { display: false } }
						}
					}
				});
			});
	</script>
	`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(tmpl))
}

// API que fornece os dados JSON extraídos da BD
func fetchStatsData(w http.ResponseWriter, r *http.Request) {
	var res StatsResponse

	// Obter estatísticas de Gravidade
	rows, err := db.Query("SELECT severity, COUNT(*) FROM syslogs GROUP BY severity ORDER BY count DESC")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var s SeverityStat
			rows.Scan(&s.Severity, &s.Count)
			res.Severities = append(res.Severities, s)
		}
	}

	// Obter Top 5 Hosts (ignorando '-' vazios)
	rows2, err := db.Query("SELECT hostname, COUNT(*) FROM syslogs WHERE hostname != '-' GROUP BY hostname ORDER BY count DESC LIMIT 5")
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var h HostStat
			rows2.Scan(&h.Hostname, &h.Count)
			res.Hosts = append(res.Hosts, h)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}