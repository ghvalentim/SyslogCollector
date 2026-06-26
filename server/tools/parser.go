package tools

import (
	"github.com/influxdata/go-syslog/v3/rfc3164"
	"github.com/influxdata/go-syslog/v3/rfc5424"
	"github.com/influxdata/go-syslog/v3"
	"time"
	"fmt"
	"strings"
	"syslog-collector/models"
)


// formatSeverity converte o código de severidade num nome legível, como "Emergência", "Alerta", "Crítico", etc.
func formatSeverity(sev uint8) string {
	switch sev { 
		case 0: return "Emergência"; 
		case 1: return "Alerta"; 
		case 2: return "Crítico"; 
		case 3: return "Erro"; 
		case 4: return "Aviso"; 
		case 5: return "Notice"; 
		case 6: return "Info"; 
		case 7: return "Debug"; 
		default: return "Info" }
}

// extractSyslogData extrai os campos relevantes de uma mensagem syslog e preenche a estrutura LogEntry.
func extractSyslogData(entry *models.LogEntry, msg syslog.Message) {
	switch m := msg.(type) {
	case *rfc5424.SyslogMessage:
		if m.Timestamp != nil { entry.Timestamp = m.Timestamp.Format(time.RFC3339) }
		if m.Hostname != nil { entry.Hostname = *m.Hostname }
		if m.Appname != nil { entry.AppName = *m.Appname }
		if m.Severity != nil { entry.Severity = formatSeverity(*m.Severity) }
		if m.Facility != nil { entry.Facility = fmt.Sprint(*m.Facility) }
		if m.Message != nil { entry.Payload = strings.TrimSpace(*m.Message) }
	case *rfc3164.SyslogMessage:
		if m.Timestamp != nil { entry.Timestamp = m.Timestamp.Format(time.RFC3339) }
		if m.Hostname != nil { entry.Hostname = *m.Hostname }
		if m.Appname != nil { entry.AppName = *m.Appname }
		if m.Severity != nil { entry.Severity = formatSeverity(*m.Severity) }
		if m.Facility != nil { entry.Facility = fmt.Sprint(*m.Facility) }
		if m.Message != nil { entry.Payload = strings.TrimSpace(*m.Message) }
	}
}

// ParseSyslog tenta analisar a mensagem syslog usando os formatos RFC5424 e RFC3164, preenchendo os campos do LogEntry.
func ParseSyslog(entry *models.LogEntry) {
    if msg, err := rfc5424.NewParser().Parse([]byte(entry.Payload)); err == nil && msg != nil {
        extractSyslogData(entry, msg)
        return
    }

    if msg, err := rfc3164.NewParser().Parse([]byte(entry.Payload)); err == nil && msg != nil {
        extractSyslogData(entry, msg)
    }
}

/* Parser.go contém funções para analisar mensagens syslog recebidas, 
extraindo campos relevantes e preenchendo a estrutura LogEntry.
As mensagens são analisadas usando os formatos RFC5424 e RFC3164, 
garantindo compatibilidade com diferentes tipos de syslog.
*/