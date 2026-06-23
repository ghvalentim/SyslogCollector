package models

import "database/sql"



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

type SeverityStat struct { Severity string `json:"severity"`; Count int `json:"count"` }
type HostStat struct { Hostname string `json:"hostname"`; Count int `json:"count"` }
type SourceStat struct { Source string `json:"source"`; Count int `json:"count"` }
type StatsResponse struct { Severities []SeverityStat `json:"severities"`; Hosts []HostStat `json:"hosts"`; Sources []SourceStat `json:"sources"` }


type Settings struct { 
	Retention int 
	User string 
	Error string
	TgBotUser string
	TgChatID string
 }

type LogPolicy struct {
	Enabled         bool     `json:"enabled"`
	MinimumSeverity string   `json:"minimum_severity"`
	IgnoredApps     []string `json:"ignored_apps"`
	IgnoredHosts    []string `json:"ignored_hosts"`
	IgnoredKeywords []string `json:"ignored_keywords"`
}

type PolicyViewData struct {
	Policy      LogPolicy
	AppsStr     string
	HostsStr    string
	KeywordsStr string
	Stats       map[string]string
}

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

type AlertNotification struct {
	RuleName      string
	Occurrences   int
	WindowMinutes int
	SampleLog     string
}

type API struct {
	Token string
	EndpointURL string
	Requests func(method, url string, body []byte) (respBody []byte, err error)
}

