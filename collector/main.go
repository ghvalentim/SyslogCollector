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

type LogEntry struct {
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

type LogPolicy struct {
	Enabled         bool     `json:"enabled"`
	MinimumSeverity string   `json:"minimum_severity"`
	IgnoredApps     []string `json:"ignored_apps"`
	IgnoredHosts    []string `json:"ignored_hosts"`
	IgnoredKeywords []string `json:"ignored_keywords"`
}

type LogJob struct {
	SourceIP string
	Protocol string
	Payload  string
}

var (
	ctx          = context.Background()
	activePolicy LogPolicy
	policyMutex  sync.RWMutex
	logChan      = make(chan LogJob, 10000) // Buffer da Worker Pool
)

func main() {
	defer func() { if r := recover(); r != nil { log.Fatalf("[CRÍTICO] Falha fatal: %v", r) } }()

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" { redisURL = "redis:6379" }
	rdb := redis.NewClient(&redis.Options{Addr: redisURL, DB: 0})
	if _, err := rdb.Ping(ctx).Result(); err != nil { log.Fatalf("Erro Redis: %v", err) }

	loadPolicyFromRedis(rdb)
	go watchPolicyUpdates(rdb)

	// INICIAR WORKER POOL (10 Workers em paralelo)
	for i := 0; i < 10; i++ {
		go workerPool(rdb)
	}

	log.Println("Collector ativo nas portas 514 (UDP e TCP), a aguardar logs...")
	go startUDPServer()
	startTCPServer()
}

func workerPool(rdb *redis.Client) {
	for job := range logChan {
		processAndQueueLog(rdb, job.SourceIP, job.Protocol, job.Payload)
	}
}

func loadPolicyFromRedis(rdb *redis.Client) {
	if val, err := rdb.Get(ctx, "active_log_policy").Result(); err == nil {
		var p LogPolicy
		if json.Unmarshal([]byte(val), &p) == nil {
			policyMutex.Lock()
			activePolicy = p
			policyMutex.Unlock()
		}
	}
}

func watchPolicyUpdates(rdb *redis.Client) {
	pubsub := rdb.Subscribe(ctx, "policy_updates")
	defer pubsub.Close()
	for range pubsub.Channel() { loadPolicyFromRedis(rdb) }
}

func startUDPServer() {
	addr := net.UDPAddr{Port: 514, IP: net.ParseIP("0.0.0.0")}
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil { log.Fatalf("Erro UDP: %v", err) }
	defer conn.Close()
	buf := make([]byte, 8192)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err == nil { logChan <- LogJob{SourceIP: remoteAddr.IP.String(), Protocol: "UDP", Payload: string(buf[:n])} }
	}
}

func startTCPServer() {
	listener, err := net.Listen("tcp", ":514")
	if err != nil { log.Fatalf("Erro TCP: %v", err) }
	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err == nil { go handleTCPConnection(conn) }
	}
}

func handleTCPConnection(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 8192)
	for {
		n, err := conn.Read(buf)
		if err != nil { break }
		ip := strings.Split(conn.RemoteAddr().String(), ":")[0]
		logChan <- LogJob{SourceIP: ip, Protocol: "TCP", Payload: string(buf[:n])}
	}
}

func ClassifySource(appName, payload string) string {
	text := strings.ToLower(appName + " " + payload)
	
	if containsAny(text, []string{"pfsense", "opnsense", "fortigate", "sophos", "watchguard", "iptables", "nftables"}) { return "Firewall" }
	if containsAny(text, []string{"microsoft", "winrm", "windows", "eventlog"}) { return "Windows" }
	if containsAny(text, []string{"systemd", "kernel", "cron", "sshd", "rsyslog"}) { return "Linux" }
	if containsAny(text, []string{"nginx", "apache", "apache2", "httpd", "iis"}) { return "Web" }
	if containsAny(text, []string{"named", "bind", "unbound"}) { return "DNS" }
	if containsAny(text, []string{"dhcpd", "kea", "dnsmasq"}) { return "DHCP" }
	if containsAny(text, []string{"cisco", "mikrotik", "routeros", "juniper", "aruba", "hp"}) { return "Network" }
	
	return "Unknown"
}

func containsAny(s string, substrs []string) bool {
	for _, sub := range substrs { if strings.Contains(s, sub) { return true } }; return false
}

func HumanizeFacility(facStr string) string {
	facMap := map[string]string{
		"0": "Kernel", "1": "User", "2": "Mail", "3": "Daemon", "4": "Auth", "5": "Syslog",
		"6": "LPR", "7": "News", "8": "UUCP", "9": "Cron", "10": "AuthPriv", "11": "FTP",
		"12": "NTP", "13": "Security", "14": "Console", "15": "Clock", "16": "Local0",
		"17": "Local1", "18": "Local2", "19": "Local3", "20": "Local4", "21": "Local5",
		"22": "Local6", "23": "Local7",
	}
	if name, exists := facMap[facStr]; exists { return name }
	return "Unknown"
}

func processAndQueueLog(rdb *redis.Client, sourceIP, protocol, rawPayload string) {
	rdb.Incr(ctx, "stats:received_total")

	entry := LogEntry{ Timestamp: time.Now().Format(time.RFC3339), SourceIP: sourceIP, Protocol: protocol, Hostname: "-", AppName: "-", Severity: "Info", Facility: "-", FacilityName: "Unknown", SourceType: "Unknown", Payload: strings.TrimSpace(rawPayload) }

	if msg, err := rfc5424.NewParser().Parse([]byte(entry.Payload)); err == nil && msg != nil {
		extractSyslogData(&entry, msg)
	} else if msg3164, err3164 := rfc3164.NewParser().Parse([]byte(entry.Payload)); err3164 == nil && msg3164 != nil {
		extractSyslogData(&entry, msg3164)
	}

	entry.SourceType = ClassifySource(entry.AppName, entry.Payload)
	entry.FacilityName = HumanizeFacility(entry.Facility)

	if !ApplyPolicies(rdb, &entry) { return }

	if jsonData, err := json.Marshal(entry); err == nil {
		rdb.LPush(ctx, "syslog_queue", jsonData)
		rdb.Incr(ctx, "stats:stored_total")
	}
}

func extractSyslogData(entry *LogEntry, msg syslog.Message) {
	switch m := msg.(type) {
	case *rfc5424.SyslogMessage:
		if m.Timestamp != nil { entry.Timestamp = m.Timestamp.Format(time.RFC3339) }
		if m.Hostname != nil { entry.Hostname = *m.Hostname }
		if m.Appname != nil { entry.AppName = *m.Appname }
		if m.Severity != nil { entry.Severity = formatSeverity(*m.Severity) }
		if m.Facility != nil { entry.Facility = fmt.Sprint(*m.Facility) }
		if m.Message != nil { entry.Payload = strings.TrimSpace(*m.Message) }
	case *rfc3164.SyslogMessage:
		if m.Timestamp != nil { entry.Timestamp = m.Timestamp.Format(time.RFC3339) }
		if m.Hostname != nil { entry.Hostname = *m.Hostname }
		if m.Appname != nil { entry.AppName = *m.Appname }
		if m.Severity != nil { entry.Severity = formatSeverity(*m.Severity) }
		if m.Facility != nil { entry.Facility = fmt.Sprint(*m.Facility) }
		if m.Message != nil { entry.Payload = strings.TrimSpace(*m.Message) }
	}
}

func ApplyPolicies(rdb *redis.Client, entry *LogEntry) bool {
	policyMutex.RLock(); p := activePolicy; policyMutex.RUnlock()
	if !p.Enabled { return true }

	sevs := map[string]int{"Emergência": 0, "Alerta": 1, "Crítico": 2, "Erro": 3, "Aviso": 4, "Notice": 5, "Info": 6, "Debug": 7}
	if sevs[entry.Severity] > sevs[p.MinimumSeverity] {
		rdb.Incr(ctx, "stats:filtered_severity"); rdb.Incr(ctx, "stats:filtered_total"); return false
	}
	for _, app := range p.IgnoredApps {
		if app != "" && strings.EqualFold(strings.TrimSpace(entry.AppName), strings.TrimSpace(app)) { rdb.Incr(ctx, "stats:filtered_app"); rdb.Incr(ctx, "stats:filtered_total"); return false }
	}
	for _, host := range p.IgnoredHosts {
		if host != "" && strings.EqualFold(strings.TrimSpace(entry.Hostname), strings.TrimSpace(host)) { rdb.Incr(ctx, "stats:filtered_host"); rdb.Incr(ctx, "stats:filtered_total"); return false }
	}
	payloadLower := strings.ToLower(entry.Payload)
	for _, kw := range p.IgnoredKeywords {
		if kw != "" && strings.Contains(payloadLower, strings.ToLower(strings.TrimSpace(kw))) { rdb.Incr(ctx, "stats:filtered_keyword"); rdb.Incr(ctx, "stats:filtered_total"); return false }
	}
	return true
}

func formatSeverity(sev uint8) string {
	switch sev { case 0: return "Emergência"; case 1: return "Alerta"; case 2: return "Crítico"; case 3: return "Erro"; case 4: return "Aviso"; case 5: return "Notice"; case 6: return "Info"; case 7: return "Debug"; default: return "Info" }
}