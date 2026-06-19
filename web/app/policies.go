package app

import (
	"net/http"
	_ "github.com/redis/go-redis/v9"
	_ "github.com/lib/pq"

)

// --- POLÍTICAS ---
func servePoliciesView(w http.ResponseWriter, r *http.Request) {
	var data PolicyViewData
	db.QueryRow("SELECT enabled, minimum_severity, ignored_apps, ignored_hosts, ignored_keywords FROM log_policies WHERE id=1").Scan(&data.Policy.Enabled, &data.Policy.MinimumSeverity, &data.AppsStr, &data.HostsStr, &data.KeywordsStr)
	data.Stats = map[string]string{ "TotalRecebidos": getRedisStat("stats:received_total"), "TotalGuardados": getRedisStat("stats:stored_total"), "TotalFiltrados": getRedisStat("stats:filtered_total"), "BySeverity": getRedisStat("stats:filtered_severity"), "ByApp": getRedisStat("stats:filtered_app"), "ByHost": getRedisStat("stats:filtered_host"), "ByKeyword": getRedisStat("stats:filtered_keyword") }
	RenderTemplate(w, "templates/policies.html", data)
}

func getRedisStat(key string) string { val, err := rdb.Get(ctx, key).Result(); if err != nil || val == "" { return "0" }; return val }

func savePolicies(w http.ResponseWriter, r *http.Request) {
	enabled := r.FormValue("enabled") == "on"
	db.Exec("UPDATE log_policies SET enabled=$1, minimum_severity=$2, ignored_apps=$3, ignored_hosts=$4, ignored_keywords=$5 WHERE id=1", enabled, r.FormValue("min_severity"), r.FormValue("apps"), r.FormValue("hosts"), r.FormValue("keywords"))
	syncPolicyToRedis()
	w.Write([]byte(`<div class="p-3 bg-emerald-50 text-emerald-700 rounded-lg text-sm flex items-center font-medium"><i data-lucide="check-circle" class="w-5 h-5 mr-2"></i> Regras atualizadas no Collector com sucesso!</div><script>lucide.createIcons();</script>`))
}