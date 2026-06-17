package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/influxdata/go-syslog/v3"
	"github.com/influxdata/go-syslog/v3/rfc3164"
	"github.com/influxdata/go-syslog/v3/rfc5424"
)

// LogEntry estrutura do log normalizado
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	SourceIP  string `json:"source_ip"`
	Protocol  string `json:"protocol"`
	Hostname  string `json:"hostname"`
	AppName   string `json:"app_name"`
	Severity  string `json:"severity"`
	Facility string `json:"facility"`
	Payload   string `json:"payload"`
}

var ctx = context.Background()

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

	log.Println("Collector ativo nas portas 514 (UDP e TCP), a aguardar logs...")

	// Inicia servidor UDP numa thread paralela
	go startUDPServer(rdb)

	// Inicia servidor TCP na thread principal
	startTCPServer(rdb)
}

func startUDPServer(rdb *redis.Client) {
	addr := net.UDPAddr{Port: 514, IP: net.ParseIP("0.0.0.0")}
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		log.Fatalf("Erro ao iniciar servidor UDP: %v", err)
	}
	defer conn.Close()

	buf := make([]byte, 8192)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		remoteIP := remoteAddr.IP.String()
		payload := string(buf[:n])
		go processAndQueueLog(rdb, remoteIP, "UDP", payload)
	}
}

func startTCPServer(rdb *redis.Client) {
	listener, err := net.Listen("tcp", ":514")
	if err != nil {
		log.Fatalf("Erro ao iniciar servidor TCP: %v", err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handleTCPConnection(rdb, conn)
	}
}

func handleTCPConnection(rdb *redis.Client, conn net.Conn) {
	defer conn.Close()
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[ERRO] Panic ao processar conexão TCP: %v\n", r)
		}
	}()

	buf := make([]byte, 8192)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			break // Fim da conexão ou erro de rede
		}

		remoteIP := strings.Split(conn.RemoteAddr().String(), ":")[0]
		payload := string(buf[:n])
		go processAndQueueLog(rdb, remoteIP, "TCP", payload)
	}
}

func processAndQueueLog(rdb *redis.Client, sourceIP, protocol, rawPayload string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[ERRO] Panic ao enfileirar log: %v\n", r)
		}
	}()

	rawPayload = strings.TrimSpace(rawPayload)

	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		SourceIP:  sourceIP,
		Protocol:  protocol,
		Hostname:  "-",
		AppName:   "-",
		Severity:  "Info",
		Payload:   rawPayload,
		Facility:  "-",
	}

	// 1. Tentar parse RFC5424 (ex: Fluent-bit, Windows WEF)
	p5424 := rfc5424.NewParser()
	msg, err := p5424.Parse([]byte(rawPayload))
	if err == nil && msg != nil {
		extractSyslogData(&entry, msg)
	} else {
		// 2. Tentar RFC3164 (equipamentos de rede mais antigos NAT/Switches)
		p3164 := rfc3164.NewParser()
		msg3164, err3164 := p3164.Parse([]byte(rawPayload))
		if err3164 == nil && msg3164 != nil {
			extractSyslogData(&entry, msg3164)
		}
	}

	jsonData, err := json.Marshal(entry)
	if err != nil {
		log.Printf("[ERRO] Falha ao converter log para JSON: %v\n", err)
		return
	}

	// Enviar para a fila Redis
	err = rdb.LPush(ctx, "syslog_queue", jsonData).Err()
	if err != nil {
		log.Printf("[ERRO] Falha ao enviar log para Redis: %v\n", err)
	}
}

// A Magia acontece aqui: Conversão de tipo segura (Type Assertion) para go-syslog/v3
func extractSyslogData(entry *LogEntry, msg syslog.Message) {
	switch m := msg.(type) {
	
	// Se for o formato mais moderno (RFC5424)
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
			entry.Facility = formatFacility(*m.Facility)
		}
		if m.Message != nil {
			entry.Payload = strings.TrimSpace(*m.Message)
		}
		
	// Se for o formato legado (RFC3164)
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
			entry.Facility = formatFacility(*m.Facility)
		}
		if m.Message != nil {
			entry.Payload = strings.TrimSpace(*m.Message)
		}
	}
}

// Traduz o nível de severidade numérico (Syslog RFC) para texto amigável
func formatSeverity(sev uint8) string {
	switch sev {
	case 0: return "Emergência"
	case 1: return "Alerta"
	case 2: return "Crítico"
	case 3: return "Erro"
	case 4: return "Aviso"
	case 5: return "Notice"
	case 6: return "Info"
	case 7: return "Debug"
	default: return "Info"
	}
}

// Traduz o nível de facility numérico (Syslog RFC) para texto amigável
func formatFacility(fac uint8) string {
	switch fac {
	case 0: return "Kernel"
	case 1: return "User"
	case 2: return "Mail"
	case 3: return "System"
	case 4: return "Security"
	case 5: return "Syslog"
	case 6: return "Printer"
	case 7: return "News"
	case 8: return "UUCP"
	case 9: return "Cron"
	case 10: return "Auth"
	case 11: return "FTP"
	case 12: return "NTP"
	case 13: return "Log Audit"
	case 14: return "Log Alert"
	case 15: return "Clock Daemon"
	default: return "Unknown"
	}
}