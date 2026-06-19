package app

import (
	"fmt"
	"time"
	"encoding/json"

	_ "github.com/redis/go-redis/v9"
	_ "github.com/lib/pq"
)


func InitServices() {
	go logWorker()
	go retentionWorker()
}


// --- WORKERS ---
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