package tools

import (
	"github.com/redis/go-redis/v9"
	"strings"
	"time"
	"encoding/json"
	"syslog-collector/models"
	"syslog-collector/settings"
)

// --- WORKER POOL DE LOGS ---
// logChan é o canal usado para enviar jobs de logs para a pool de workers.
var logChan = make(chan models.LogJob, 10000) // Buffer da Worker Pool
var ctx = settings.GetContext()

// InitWorker inicializa a pool de workers que processam logs recebidos do listener e os armazenam no Redis.
func InitWorker(rdb *redis.Client) {
	for i := 0; i < 10; i++ {
		go workerPool(rdb)
	}
}

//Worker Pool que processa logs recebidos do listener.
func workerPool(rdb *redis.Client) {
	for job := range logChan {
		processAndQueueLog(rdb, job.SourceIP, job.Protocol, job.Payload)
	}
}

// processAndQueueLog processa um log recebido, aplicando parsing, classificação, políticas, e se aprovado, armazena no Redis.
func processAndQueueLog(rdb *redis.Client, sourceIP, protocol, rawPayload string) {
	rdb.Incr(ctx, "stats:received_total")

	entry := models.LogEntry{ Timestamp: time.Now().Format(time.RFC3339), SourceIP: sourceIP, Protocol: protocol, Hostname: "-", AppName: "-", Severity: "Info", Facility: "-", FacilityName: "Unknown", SourceType: "Unknown", Payload: strings.TrimSpace(rawPayload) }

	ParseSyslog(&entry)

	entry.SourceType = ClassifySource(entry.AppName, entry.Payload)
	entry.FacilityName = settings.HumanizeFacility(entry.Facility)

	if !settings.ApplyPolicies(rdb, &entry) { return }

	if jsonData, err := json.Marshal(entry); err == nil {
		rdb.LPush(ctx, "syslog_queue", jsonData)
		rdb.Incr(ctx, "stats:stored_total")
	}
}

/* Worker Pool que processa logs recebidos do listener.
Cada worker lê do canal logChan, faz o parsing, classificação, aplicação de políticas, e se aprovado, armazena o log no Redis.
O número de workers pode ser ajustado conforme a carga esperada. */