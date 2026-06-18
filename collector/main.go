package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/influxdata/go-syslog/v3"
	"github.com/influxdata/go-syslog/v3/rfc3164"
	"github.com/influxdata/go-syslog/v3/rfc5424"
	"github.com/redis/go-redis/v9"
)

// Estruturas de Dados
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	SourceIP  string `json:"source_ip"`
	Protocol  string `json:"protocol"`
	Hostname  string `json:"hostname"`
	AppName   string `json:"app_name"`
	Severity  string `json:"severity"`
	Facility  string `json:"facility"`
	Payload   string `json:"payload"`
}

// Estrutura da Política de Logs
type LogPolicy struct {
	Enabled         bool     `json:"enabled"`
	MinimumSeverity string   `json:"minimum_severity"`
	IgnoredApps     []string `json:"ignored_apps"`
	IgnoredHosts    []string `json:"ignored_hosts"`
	IgnoredKeywords []string `json:"ignored_keywords"`
}

var (
	ctx          = context.Background()
	activePolicy LogPolicy
	policyMutex  sync.RWMutex
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Fatalf("[CRÍTICO] Falha fatal no Collector: %v", r)
		}
	}()

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis:6379"
	}

	rdb := redis.NewClient(&redis.Options{Addr: redisURL, DB: 0})
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		log.Fatalf("[ERRO] Falha ao conectar ao Redis: %v", err)
	}

	// 1. Carregar Política Inicial e Iniciar Escuta de Atualizações (Pub/Sub)
	loadPolicyFromRedis(rdb)
	go watchPolicyUpdates(rdb)

	log.Println("Collector ativo nas portas 514 (UDP e TCP), a aguardar logs...")

	// 2. Inicia servidores
	go startUDPServer(rdb)
	startTCPServer(rdb)
}

// --- MOTOR DE POLÍTICAS DE LOGS ---

func loadPolicyFromRedis(rdb *redis.Client) {
	val, err := rdb.Get(ctx, "active_log_policy").Result()
	if err == nil {
		var p LogPolicy
		if err := json.Unmarshal([]byte(val), &p); err == nil {
			policyMutex.Lock()
			activePolicy = p
			policyMutex.Unlock()
			log.Println("[POLÍTICAS] Motor de regras atualizado com sucesso via Redis.")
		}
	}
}

func watchPolicyUpdates(rdb *redis.Client) {
	pubsub := rdb.Subscribe(ctx, "policy_updates")
	defer pubsub.Close()

	// Fica bloqueado à escuta de mensagens neste canal
	for range pubsub.Channel() {
		loadPolicyFromRedis(rdb)
	}
}

func ApplyPolicies(rdb *redis.Client, entry *LogEntry) bool {
	policyMutex.RLock()
	p := activePolicy
	policyMutex.RUnlock()

	// Se o motor estiver desligado, aceita tudo
	if !p.Enabled {
		return true
	}

	// 1. Regra: Severidade Mínima
	sevs := map[string]int{"Emergência": 0, "Alerta": 1, "Crítico": 2, "Erro": 3, "Aviso": 4, "Notice": 5, "Info": 6, "Debug": 7}
	if sevs[entry.Severity] > sevs[p.MinimumSeverity] {
		rdb.Incr(ctx, "stats:filtered_severity")
		rdb.Incr(ctx, "stats:filtered_total")
		return false // Descarta
	}

	// 2. Regra: Aplicações Ignoradas
	for _, app := range p.IgnoredApps {
		if app != "" && strings.EqualFold(strings.TrimSpace(entry.AppName), strings.TrimSpace(app)) {
			rdb.Incr(ctx, "stats:filtered_app")
			rdb.Incr(ctx, "stats:filtered_total")
			return false
		}
	}

	// 3. Regra: Hosts Ignorados
	for _, host := range p.IgnoredHosts {
		if host != "" && strings.EqualFold(strings.TrimSpace(entry.Hostname), strings.TrimSpace(host)) {
			rdb.Incr(ctx, "stats:filtered_host")
			rdb.Incr(ctx, "stats:filtered_total")
			return false
		}
	}

	// 4. Regra: Palavras-chave Ignoradas
	payloadLower := strings.ToLower(entry.Payload)
	for _, kw := range p.IgnoredKeywords {
		if kw != "" && strings.Contains(payloadLower, strings.ToLower(strings.TrimSpace(kw))) {
			rdb.Incr(ctx, "stats:filtered_keyword")
			rdb.Incr(ctx, "stats:filtered_total")
			return false
		}
	}

	// Se sobreviveu a todos os filtros, é para armazenar!
	return true
}

// --- SERVIDORES DE REDE ---

func startUDPServer(rdb *redis.Client) {
	addr := net.UDPAddr{Port: 514, IP: net.ParseIP("0.0.0.0")}
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		log.Fatalf("Erro UDP: %v", err)
	}
	defer conn.Close()

	buf := make([]byte, 8192)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err == nil {
			go processAndQueueLog(rdb, remoteAddr.IP.String(), "UDP", string(buf[:n]))
		}
	}
}

func startTCPServer(rdb *redis.Client) {
	listener, err := net.Listen("tcp", ":514")
	if err != nil {
		log.Fatalf("Erro TCP: %v", err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err == nil {
			go handleTCPConnection(rdb, conn)
		}
	}
}

func handleTCPConnection(rdb *redis.Client, conn net.Conn) {
	defer conn.Close()
	defer func() { recover() }()

	buf := make([]byte, 8192)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			break
		}
		remoteIP := strings.Split(conn.RemoteAddr().String(), ":")[0]
		go processAndQueueLog(rdb, remoteIP, "TCP", string(buf[:n]))
	}
}

// --- PROCESSAMENTO PRINCIPAL ---

func processAndQueueLog(rdb *redis.Client, sourceIP, protocol, rawPayload string) {
	defer func() { recover() }()
	rdb.Incr(ctx, "stats:received_total") // Incrementa contador de entrada

	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		SourceIP:  sourceIP,
		Protocol:  protocol,
		Hostname:  "-",
		AppName:   "-",
		Severity:  "Info",
		Facility:  "-",
		Payload:   strings.TrimSpace(rawPayload),
	}

	// Parsing RFC5424 / RFC3164
	p5424 := rfc5424.NewParser()
	if msg, err := p5424.Parse([]byte(entry.Payload)); err == nil && msg != nil {
		extractSyslogData(&entry, msg)
	} else {
		p3164 := rfc3164.NewParser()
		if msg3164, err3164 := p3164.Parse([]byte(entry.Payload)); err3164 == nil && msg3164 != nil {
			extractSyslogData(&entry, msg3164)
		}
	}

	// **********************************************
	// INTEGRAÇÃO DO MOTOR DE POLÍTICAS (ROADMAP - PONTO 8)
	// **********************************************
	if !ApplyPolicies(rdb, &entry) {
		return // Descartado pelas regras!
	}

	// Aprovado: Envia para a Fila do Redis
	jsonData, err := json.Marshal(entry)
	if err == nil {
		rdb.LPush(ctx, "syslog_queue", jsonData)
		rdb.Incr(ctx, "stats:stored_total") // Incrementa contador de armazenados
	}
}

func extractSyslogData(entry *LogEntry, msg syslog.Message) {
	switch m := msg.(type) {
	case *rfc5424.SyslogMessage:
		if m.Timestamp != nil {
			entry.Timestamp = m.Timestamp.Format(time.RFC3339)
		}
		if m.Hostname != nil {
			entry.Hostname = *m.Hostname
		}
		if m.Appname != nil {
			entry.AppName = *m.Appname
		}
		if m.Severity != nil {
			entry.Severity = formatSeverity(*m.Severity)
		}
		if m.Facility != nil {
			entry.Facility = fmt.Sprint(*m.Facility)
		}
		if m.Message != nil {
			entry.Payload = strings.TrimSpace(*m.Message)
		}
	case *rfc3164.SyslogMessage:
		if m.Timestamp != nil {
			entry.Timestamp = m.Timestamp.Format(time.RFC3339)
		}
		if m.Hostname != nil {
			entry.Hostname = *m.Hostname
		}
		if m.Appname != nil {
			entry.AppName = *m.Appname
		}
		if m.Severity != nil {
			entry.Severity = formatSeverity(*m.Severity)
		}
		if m.Facility != nil {
			entry.Facility = fmt.Sprint(*m.Facility)
		}
		if m.Message != nil {
			entry.Payload = strings.TrimSpace(*m.Message)
		}
	}
}

func formatSeverity(sev uint8) string {
	switch sev {
	case 0:
		return "Emergência"
	case 1:
		return "Alerta"
	case 2:
		return "Crítico"
	case 3:
		return "Erro"
	case 4:
		return "Aviso"
	case 5:
		return "Notice"
	case 6:
		return "Info"
	case 7:
		return "Debug"
	default:
		return "Info"
	}
}
