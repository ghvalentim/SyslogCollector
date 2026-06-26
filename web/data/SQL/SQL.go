package SQL

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"syslog-web/data/Redis"
	"time"
	"strings"
	"syslog-web/models"
	"encoding/json"
	"log"
	_ "github.com/lib/pq"
)

var (
	DB  = getDB()
	Ctx = getContext()
)

// InitSQL inicializa a base de dados PostgreSQL, criando tabelas e colunas necessárias se não existirem, e sincroniza a política de logs para o Redis.
func InitSQL() {
	for i := 0; i < 5; i++ {
		if err := DB.Ping(); err != nil {
			log.Printf("Tentativa %d: Erro ao conectar à base de dados PostgreSQL: %v", i+1, err)
			time.Sleep(2 * time.Second)
		} else {
			log.Println("Conexão com a base de dados PostgreSQL estabelecida com sucesso.")
			break
		}
		if i == 4 {
			log.Fatalf("Falha ao conectar à base de dados PostgreSQL após 5 tentativas.")
		}
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

func getContext() context.Context {
	Ctx := context.Background()
	return Ctx
}

func getDB() *sql.DB {
	connStr := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable", os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASS"), os.Getenv("DB_NAME"))
	DB, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Erro ao conectar à base de dados PostgreSQL: %v", err)
	}
	return DB
}

// --- POLÍTICA DE LOGS ---
// SyncPolicyToRedis sincroniza a política de logs da base de dados PostgreSQL para o Redis, permitindo que os workers acedam às definições atualizadas.
func SyncPolicyToRedis() {
	var enabled bool; var minSev, appsStr, hostsStr, kwStr string
	DB.QueryRow("SELECT enabled, minimum_severity, ignored_apps, ignored_hosts, ignored_keywords FROM log_policies WHERE id=1").Scan(&enabled, &minSev, &appsStr, &hostsStr, &kwStr)
	policy := models.LogPolicy{ Enabled: enabled, MinimumSeverity: minSev, IgnoredApps: parseList(appsStr), IgnoredHosts: parseList(hostsStr), IgnoredKeywords: parseList(kwStr) }
	jsonData, _ := json.Marshal(policy)
	Redis.Rdb.Set(Redis.Ctx, "active_log_policy", jsonData, 0)
	Redis.Rdb.Publish(Redis.Ctx, "policy_updates", "reload")
}

func parseList(i string) []string { p := strings.Split(i, ","); var r []string; for _, x := range p { if t := strings.TrimSpace(x); t != "" { r = append(r, t) } }; return r }