package models

import "database/sql"



// LogEntry representa uma entrada de log syslog armazenada na base de dados.
type LogEntry struct {
	ID           int    `json:"id"`
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

// SeverityStat representa a contagem de logs por nível de severidade.
type SeverityStat struct { Severity string `json:"severity"`; Count int `json:"count"` }
// HostStat representa a contagem de logs por hostname.
type HostStat struct { Hostname string `json:"hostname"`; Count int `json:"count"` }
// SourceStat representa a contagem de logs por tipo de origem.
type SourceStat struct { Source string `json:"source"`; Count int `json:"count"` }
// StatsResponse encapsula as estatísticas agregadas de logs, incluindo contagens por severidade, host e tipo de origem.
type StatsResponse struct { Severities []SeverityStat `json:"severities"`; Hosts []HostStat `json:"hosts"`; Sources []SourceStat `json:"sources"` }


// Settings representa as definições da aplicação, incluindo política de retenção de logs e informações do administrador.
type Settings struct { 
	Retention int 
	User string 
	Error string
	TgBotUser string
	TgChatID string
	NotifyTelegram bool
	NotifyEmail bool
 }

 // LogPolicy representa a política de logs configurada pelo utilizador, 
 // incluindo critérios de severidade mínima e listas de aplicações, hosts e palavras-chave ignoradas.
type LogPolicy struct {
	Enabled         bool     `json:"enabled"`
	MinimumSeverity string   `json:"minimum_severity"`
	IgnoredApps     []string `json:"ignored_apps"`
	IgnoredHosts    []string `json:"ignored_hosts"`
	IgnoredKeywords []string `json:"ignored_keywords"`
}

// PolicyViewData encapsula os dados necessários para renderizar a página de visualização da política de logs, 
// incluindo a política atual, listas de aplicações, hosts e palavras-chave ignoradas, 
// bem como estatísticas relacionadas.
type PolicyViewData struct {
	Policy      LogPolicy
	AppsStr     string
	HostsStr    string
	KeywordsStr string
	Stats       map[string]string
}

// AlertRule representa uma regra de alerta configurada pelo utilizador, 
// incluindo critérios de severidade, tipo de origem, palavra-chave, limiar e janela temporal.
type AlertRule struct {
	ID            int
	Enabled       bool
	Name          string
	Severity      string
	SourceType    string
	Keyword       string
	Threshold     int
	WindowMinutes int
	LastTriggered  sql.NullTime
}




