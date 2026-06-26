package api

import (
	"net/http"
	"syslog-web/app/tools"
)

// APIStats é o handler da API que retorna estatísticas agregadas de logs em formato JSON.

func APIStats(w http.ResponseWriter, r *http.Request) {
	tools.FetchStatsData(w, r)
}