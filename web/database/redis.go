package database

import (
	"encoding/json"
	"log"
	"os"
	"strings"
	"github.com/redis/go-redis/v9"
	_ "github.com/lib/pq"
	"syslog-web/models"
	
)


// --- REDIS ---
// InitRedis inicializa o cliente Redis e verifica a conectividade.
func InitRedis() {
	Rdb = redis.NewClient(&redis.Options{Addr: os.Getenv("REDIS_URL"), DB: 0})
	if _, err := Rdb.Ping(Ctx).Result(); err != nil {
		log.Fatalf("Erro Redis: %v", err)
	}
}

// --- POLÍTICA DE LOGS ---
// SyncPolicyToRedis sincroniza a política de logs da base de dados PostgreSQL para o Redis, permitindo que os workers acedam às definições atualizadas.
func SyncPolicyToRedis() {
	var enabled bool; var minSev, appsStr, hostsStr, kwStr string
	DB.QueryRow("SELECT enabled, minimum_severity, ignored_apps, ignored_hosts, ignored_keywords FROM log_policies WHERE id=1").Scan(&enabled, &minSev, &appsStr, &hostsStr, &kwStr)
	policy := models.LogPolicy{ Enabled: enabled, MinimumSeverity: minSev, IgnoredApps: parseList(appsStr), IgnoredHosts: parseList(hostsStr), IgnoredKeywords: parseList(kwStr) }
	jsonData, _ := json.Marshal(policy)
	Rdb.Set(Ctx, "active_log_policy", jsonData, 0)
	Rdb.Publish(Ctx, "policy_updates", "reload")
}

func parseList(i string) []string { p := strings.Split(i, ","); var r []string; for _, x := range p { if t := strings.TrimSpace(x); t != "" { r = append(r, t) } }; return r }