package app

import (
	"github.com/redis/go-redis/v9"
	"strings"
	"time"
	"encoding/json"
)

var logChan = make(chan LogJob, 10000) // Buffer da Worker Pool

func InitWorker(rdb *redis.Client) {
	for i := 0; i < 10; i++ {
		go workerPool(rdb)
	}
}

func workerPool(rdb *redis.Client) {
	for job := range logChan {
		processAndQueueLog(rdb, job.SourceIP, job.Protocol, job.Payload)
	}
}

func processAndQueueLog(rdb *redis.Client, sourceIP, protocol, rawPayload string) {
	rdb.Incr(ctx, "stats:received_total")

	entry := LogEntry{ Timestamp: time.Now().Format(time.RFC3339), SourceIP: sourceIP, Protocol: protocol, Hostname: "-", AppName: "-", Severity: "Info", Facility: "-", FacilityName: "Unknown", SourceType: "Unknown", Payload: strings.TrimSpace(rawPayload) }

	ParseSyslog(&entry)

	entry.SourceType = ClassifySource(entry.AppName, entry.Payload)
	entry.FacilityName = HumanizeFacility(entry.Facility)

	if !ApplyPolicies(rdb, &entry) { return }

	if jsonData, err := json.Marshal(entry); err == nil {
		rdb.LPush(ctx, "syslog_queue", jsonData)
		rdb.Incr(ctx, "stats:stored_total")
	}
}

/* Worker Pool que processa logs recebidos do listener.
Cada worker lê do canal logChan, faz o parsing, classificação, aplicação de políticas, e se aprovado, armazena o log no Redis.
O número de workers pode ser ajustado conforme a carga esperada. */