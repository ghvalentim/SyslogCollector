package app

type LogEntry struct {
	Timestamp    string `json:"timestamp"`
	SourceIP     string `json:"source_ip"`
	Protocol     string `json:"protocol"`
	Hostname     string `json:"hostname"`
	AppName      string `json:"app_name"`
	Severity     string `json:"severity"`
	Facility     string `json:"facility"`
	FacilityName string `json:"facility_name"`
	SourceType   string `json:"source_type"`
	Payload      string `json:"payload"`
}

type LogPolicy struct {
	Enabled         bool     `json:"enabled"`
	MinimumSeverity string   `json:"minimum_severity"`
	IgnoredApps     []string `json:"ignored_apps"`
	IgnoredHosts    []string `json:"ignored_hosts"`
	IgnoredKeywords []string `json:"ignored_keywords"`
}

type LogJob struct {
	SourceIP string
	Protocol string
	Payload  string
}

/* Modelos de dados para logs, políticas, e jobs.
LogEntry representa um log processado, com campos extraídos e classificados.
LogPolicy define as regras de filtragem ativas, que podem ser atualizadas dinamicamente.
LogJob é a estrutura básica para transportar dados brutos do listener para a worker pool. */