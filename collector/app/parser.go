package app

import (
	"github.com/influxdata/go-syslog/v3/rfc3164"
	"github.com/influxdata/go-syslog/v3/rfc5424"
	"github.com/influxdata/go-syslog/v3"
	"time"
	"fmt"
	"strings"
)

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

func extractSyslogData(entry *LogEntry, msg syslog.Message) {
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

func ParseSyslog(entry *LogEntry) {
    if msg, err := rfc5424.NewParser().Parse([]byte(entry.Payload)); err == nil && msg != nil {
        extractSyslogData(entry, msg)
        return
    }

    if msg, err := rfc3164.NewParser().Parse([]byte(entry.Payload)); err == nil && msg != nil {
        extractSyslogData(entry, msg)
    }
}

/* Função de parsing que tenta primeiro o formato RFC5424, e se falhar, tenta o RFC3164.
Ela preenche os campos do LogEntry com os dados extraídos, 
e pode ser expandida para suportar outros formatos ou campos personalizados. */