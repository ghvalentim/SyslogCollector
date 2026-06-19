package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

var (
	db  *sql.DB
	rdb *redis.Client
	ctx = context.Background()
)

func InitData() {
	initRedis()
	initSQL()
}


func initRedis() {
	rdb := redis.NewClient(&redis.Options{Addr: os.Getenv("REDIS_URL"), DB: 0})
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		log.Fatalf("Erro Redis: %v", err)
	}
}

func initSQL() {
	connStr := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable", os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASS"), os.Getenv("DB_NAME"))
	for i := 0; i < 5; i++ { db, _ = sql.Open("postgres", connStr); if db.Ping() == nil { break }; time.Sleep(3 * time.Second) }
	
	db.Exec(`CREATE TABLE IF NOT EXISTS syslogs (id SERIAL PRIMARY KEY, timestamp TIMESTAMP, source_ip VARCHAR(50), protocol VARCHAR(10), hostname VARCHAR(100) DEFAULT '-', app_name VARCHAR(100) DEFAULT '-', severity VARCHAR(20) DEFAULT 'Info', facility VARCHAR(100) DEFAULT '-', facility_name VARCHAR(50) DEFAULT 'Unknown', source_type VARCHAR(50) DEFAULT 'Unknown', payload TEXT, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP); CREATE INDEX IF NOT EXISTS idx_timestamp ON syslogs(timestamp DESC);`)
	db.Exec("ALTER TABLE syslogs ADD COLUMN IF NOT EXISTS source_type VARCHAR(50) DEFAULT 'Unknown'")
	db.Exec("ALTER TABLE syslogs ADD COLUMN IF NOT EXISTS facility_name VARCHAR(50) DEFAULT 'Unknown'")

	db.Exec(`CREATE TABLE IF NOT EXISTS settings (id SERIAL PRIMARY KEY, retention_days INT DEFAULT 30, admin_user VARCHAR(100) DEFAULT 'admin', admin_pass VARCHAR(100) DEFAULT 'admin');`)
	db.Exec(`INSERT INTO settings (id) SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM settings WHERE id = 1)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS log_policies (id SERIAL PRIMARY KEY, enabled BOOLEAN DEFAULT false, minimum_severity VARCHAR(20) DEFAULT 'Info', ignored_apps TEXT DEFAULT '', ignored_hosts TEXT DEFAULT '', ignored_keywords TEXT DEFAULT '');`)
	db.Exec(`INSERT INTO log_policies (id) SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM log_policies WHERE id = 1)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS alert_rules (id SERIAL PRIMARY KEY, enabled BOOLEAN DEFAULT true, name VARCHAR(100), severity VARCHAR(20), source_type VARCHAR(50), keyword VARCHAR(100), threshold INT, window_minutes INT, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP);`)

	syncPolicyToRedis()
}

func syncPolicyToRedis() {
	var enabled bool; var minSev, appsStr, hostsStr, kwStr string
	db.QueryRow("SELECT enabled, minimum_severity, ignored_apps, ignored_hosts, ignored_keywords FROM log_policies WHERE id=1").Scan(&enabled, &minSev, &appsStr, &hostsStr, &kwStr)
	policy := LogPolicy{ Enabled: enabled, MinimumSeverity: minSev, IgnoredApps: parseList(appsStr), IgnoredHosts: parseList(hostsStr), IgnoredKeywords: parseList(kwStr) }
	jsonData, _ := json.Marshal(policy)
	rdb.Set(ctx, "active_log_policy", jsonData, 0)
	rdb.Publish(ctx, "policy_updates", "reload")
}

func parseList(i string) []string { p := strings.Split(i, ","); var r []string; for _, x := range p { if t := strings.TrimSpace(x); t != "" { r = append(r, t) } }; return r }
