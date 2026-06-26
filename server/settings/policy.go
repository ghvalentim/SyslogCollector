package settings

import (
	"encoding/json"
	"github.com/redis/go-redis/v9"
	"strings"
	"context"
	"sync"
	"syslog-collector/models"
)

// --- POLÍTICA DE FILTRAGEM DE LOGS ---

// ctx é o contexto global usado para operações Redis.
var ctx = GetContext()
// activePolicy mantém a política de logs atualmente ativa, carregada do Redis.
var activePolicy models.LogPolicy
// policyMutex protege o acesso à activePolicy para leituras e escritas concorrentes.
var policyMutex sync.RWMutex


// InitPolicies inicializa a política de logs, carregando-a do Redis e configurando um watcher para atualizações em tempo real.
func InitPolicies(rdb *redis.Client) {
	loadPolicyFromRedis(rdb)
	go watchPolicyUpdates(rdb)
}

func GetContext() context.Context {
	ctx := context.Background()
	return ctx
}

// loadPolicyFromRedis carrega a política de logs do Redis e atualiza a variável activePolicy.
func loadPolicyFromRedis(rdb *redis.Client) {
	if val, err := rdb.Get(ctx, "active_log_policy").Result(); err == nil {
		var p models.LogPolicy
		if json.Unmarshal([]byte(val), &p) == nil {
			policyMutex.Lock()
			activePolicy = p
			policyMutex.Unlock()
		}
	}
}

// watchPolicyUpdates assina o canal de atualizações de política no Redis e recarrega a política sempre que uma atualização é publicada.
func watchPolicyUpdates(rdb *redis.Client) {
	pubsub := rdb.Subscribe(ctx, "policy_updates")
	defer pubsub.Close()
	for range pubsub.Channel() { loadPolicyFromRedis(rdb) }
}

// ApplyPolicies aplica a política de logs ao LogEntry fornecido, retornando true se o log deve ser armazenado, ou false se deve ser descartado.
func ApplyPolicies(rdb *redis.Client, entry *models.LogEntry) bool {
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

/* Módulo de políticas de filtragem, que define regras dinâmicas para descartar logs indesejados.
As políticas são carregadas do Redis e podem ser atualizadas em tempo real através de um canal de publicação/assinatura.
A função ApplyPolicies é chamada para cada log processado, 
e decide se ele deve ser armazenado ou descartado com base nas regras ativas. */