package logs

import (
	"fmt"
	"time"
	"log"
	"encoding/json"
	"syslog-web/models"
	_ "github.com/redis/go-redis/v9"
	_ "github.com/lib/pq"
	"syslog-web/data/SQL"
	"syslog-web/data/Redis"
)

var (
	RedisClient = Redis.Rdb
	Ctx = Redis.Ctx
)

// InitServices inicializa os workers de processamento de logs e retenção de dados.
func InitWorkers() {
	go logWorker()
	go retentionWorker()

}

// --- WORKERS ---
// logWorker processa logs recebidos do Redis e os insere na base de dados PostgreSQL.
func logWorker() {
	defer func() { if r := recover(); r != nil { time.Sleep(2 * time.Second); go logWorker() } }()
	for {
		res, err := Redis.Rdb.BRPop(SQL.Ctx, 0, "syslog_queue").Result(); if err != nil { time.Sleep(1 * time.Second); continue }
		var e models.LogEntry; if json.Unmarshal([]byte(res[1]), &e) != nil { continue }
		
		_, err = SQL.DB.Exec(`INSERT INTO syslogs (timestamp, source_ip, protocol, hostname, app_name, severity, facility, facility_name, source_type, payload) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`, e.Timestamp, e.SourceIP, e.Protocol, e.Hostname, e.AppName, e.Severity, e.Facility, e.FacilityName, e.SourceType, e.Payload)
		if err != nil { Redis.Rdb.LPush(Redis.Ctx, "syslog_queue", res[1]); time.Sleep(2 * time.Second)
		log.Printf("Erro ao inserir log na base de dados: %v. Log retornado para a fila.", err)
	 } 
	}

}

// retentionWorker remove logs antigos da base de dados com base na política de retenção configurada.
func retentionWorker() {
	for {
		var days int
		if SQL.DB.QueryRow("SELECT retention_days FROM settings WHERE id = 1").Scan(&days) == nil && days > 0 { SQL.DB.Exec(fmt.Sprintf("DELETE FROM syslogs WHERE timestamp < NOW() - INTERVAL '%d days'", days)) }
		time.Sleep(1 * time.Hour)
	}

}