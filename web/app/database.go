package app

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

var (
	DB  *sql.DB
	rdb *redis.Client
	ctx = context.Background()
)

func InitData() {
	InitRedis()
	InitSQL()
}

func InitSQL() {
	connStr := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable", os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASS"), os.Getenv("DB_NAME"))
	for i := 0; i < 5; i++ {
		DB, _ = sql.Open("postgres", connStr)
		if DB.Ping() == nil {
			break
		}
		time.Sleep(3 * time.Second)
	}

	// Tabelas base
	DB.Exec(`CREATE TABLE IF NOT EXISTS syslogs (id SERIAL PRIMARY KEY, timestamp TIMESTAMP, source_ip VARCHAR(50), protocol VARCHAR(10), hostname VARCHAR(100) DEFAULT '-', app_name VARCHAR(100) DEFAULT '-', severity VARCHAR(20) DEFAULT 'Info', facility VARCHAR(100) DEFAULT '-', facility_name VARCHAR(50) DEFAULT 'Unknown', source_type VARCHAR(50) DEFAULT 'Unknown', payload TEXT, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP); CREATE INDEX IF NOT EXISTS idx_timestamp ON syslogs(timestamp DESC);`)
	DB.Exec("ALTER TABLE syslogs ADD COLUMN IF NOT EXISTS source_type VARCHAR(50) DEFAULT 'Unknown'")
	DB.Exec("ALTER TABLE syslogs ADD COLUMN IF NOT EXISTS facility_name VARCHAR(50) DEFAULT 'Unknown'")

	// Tabela Settings (Agora com suporte para Telegram)
	DB.Exec(`CREATE TABLE IF NOT EXISTS settings (id SERIAL PRIMARY KEY, retention_days INT DEFAULT 30, admin_user VARCHAR(100) DEFAULT 'admin', admin_pass VARCHAR(100) DEFAULT 'admin', tg_bot_token VARCHAR(200) DEFAULT '', tg_chat_id VARCHAR(100) DEFAULT '');`)
	DB.Exec("ALTER TABLE settings ADD COLUMN IF NOT EXISTS tg_bot_token VARCHAR(200) DEFAULT ''")
	DB.Exec("ALTER TABLE settings ADD COLUMN IF NOT EXISTS tg_chat_id VARCHAR(100) DEFAULT ''")
	DB.Exec(`INSERT INTO settings (id) SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM settings WHERE id = 1)`)

	// Tabela Log Policies
	DB.Exec(`CREATE TABLE IF NOT EXISTS log_policies (id SERIAL PRIMARY KEY, enabled BOOLEAN DEFAULT false, minimum_severity VARCHAR(20) DEFAULT 'Info', ignored_apps TEXT DEFAULT '', ignored_hosts TEXT DEFAULT '', ignored_keywords TEXT DEFAULT '');`)
	DB.Exec(`INSERT INTO log_policies (id) SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM log_policies WHERE id = 1)`)

	// Tabela Alert Rules (Agora com last_triggered)
	DB.Exec(`CREATE TABLE IF NOT EXISTS alert_rules (id SERIAL PRIMARY KEY, enabled BOOLEAN DEFAULT true, name VARCHAR(100), severity VARCHAR(20), source_type VARCHAR(50), keyword VARCHAR(100), threshold INT, window_minutes INT, last_triggered TIMESTAMP NULL, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP);`)
	DB.Exec("ALTER TABLE alert_rules ADD COLUMN IF NOT EXISTS last_triggered TIMESTAMP NULL")

	SyncPolicyToRedis()	
}



