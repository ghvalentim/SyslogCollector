package app

import (
	"encoding/json"
	"log"
	"os"
	"strings"
	"github.com/redis/go-redis/v9"
	_ "github.com/lib/pq"
)


func InitRedis() {
	rdb = redis.NewClient(&redis.Options{Addr: os.Getenv("REDIS_URL"), DB: 0})
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		log.Fatalf("Erro Redis: %v", err)
	}
}

func SyncPolicyToRedis() {
	var enabled bool; var minSev, appsStr, hostsStr, kwStr string
	DB.QueryRow("SELECT enabled, minimum_severity, ignored_apps, ignored_hosts, ignored_keywords FROM log_policies WHERE id=1").Scan(&enabled, &minSev, &appsStr, &hostsStr, &kwStr)
	policy := LogPolicy{ Enabled: enabled, MinimumSeverity: minSev, IgnoredApps: parseList(appsStr), IgnoredHosts: parseList(hostsStr), IgnoredKeywords: parseList(kwStr) }
	jsonData, _ := json.Marshal(policy)
	rdb.Set(ctx, "active_log_policy", jsonData, 0)
	rdb.Publish(ctx, "policy_updates", "reload")
}

func parseList(i string) []string { p := strings.Split(i, ","); var r []string; for _, x := range p { if t := strings.TrimSpace(x); t != "" { r = append(r, t) } }; return r }