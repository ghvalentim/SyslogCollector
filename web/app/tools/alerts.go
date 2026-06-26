package tools

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
	"syslog-web/app/services/notifier"
	"syslog-web/app/helpers"
	SQL "syslog-web/data/SQL"
	"syslog-web/models"
)

// Inicializa o motor de alertas em background
func InitAlerts() {
	go StartAlertEngine()
}

// inicia a verificação periódica de regras (Corre a cada 30 segundos)
func StartAlertEngine() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		evaluateAlertRules()
	}
}

// verifica todas as regras de alerta ativas e dispara notificações se necessário
func evaluateAlertRules() {
	if SQL.DB == nil {
		return
	}

	// 1. Configura os canais de notificação (Telegram, Email, etc.)
	var notifiers []notifier.AlertNotifier

	// ==========================================
	// 1.1 Configuração Telegram
	// ==========================================
	tgToken := os.Getenv("TG_BOT_TOKEN")
	var tgChat string
	err := SQL.DB.QueryRow("SELECT tg_chat_id FROM settings WHERE id = 1").Scan(&tgChat)
	if err == nil && tgToken != "" && tgChat != "" && tgToken != "coloque_aqui_o_seu_token_do_botfather" {
		notifiers = append(notifiers, notifier.NewTelegramNotifier(tgChat))
	}

	// ==========================================
	// 1.2 Configuração Email (SMTP)
	// ==========================================
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")
	smtpFrom := os.Getenv("SMTP_FROM")
	smtpTo := os.Getenv("SMTP_TO")

	if smtpHost != "" && smtpPort != "" && smtpTo != "" {
		notifiers = append(notifiers, notifier.NewEmailNotifier(smtpHost, smtpPort, smtpUser, smtpPass, smtpFrom, smtpTo))
	}

	// Se não configurou nem o Telegram nem o Email, não há canais de destino, logo aborta.
	if len(notifiers) == 0 {
		return 
	}

	// Agrega todos os canais configurados numa única interface
	multiNotifier := &notifier.MultiNotifier{Notifiers: notifiers}

	// ==========================================
	// 2. Avaliação de Regras Ativas
	// ==========================================
	rows, err := SQL.DB.Query("SELECT id, name, severity, source_type, keyword, threshold, window_minutes, last_triggered FROM alert_rules WHERE enabled = true")
	if err != nil {
		log.Printf("[ERRO ALERTA] Falha a ler regras da base de dados: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var rule models.AlertRule
		if err := rows.Scan(&rule.ID, &rule.Name, &rule.Severity, &rule.SourceType, &rule.Keyword, &rule.Threshold, &rule.WindowMinutes, &rule.LastTriggered); err != nil {
			continue
		}

		// Prevenir spam: Só permite disparar a mesma regra após o tempo da janela ter passado novamente
		if rule.LastTriggered.Valid && time.Since(rule.LastTriggered.Time) < time.Duration(rule.WindowMinutes)*time.Minute {
			continue 
		}

		// 3. Contar ocorrências na base de dados
		countQuery := "SELECT COUNT(*), MAX(payload) FROM syslogs WHERE timestamp >= NOW() - INTERVAL '1 minute' * $1"
		var args []interface{}
		args = append(args, rule.WindowMinutes)
		argId := 2

		if rule.Severity != "" {
			countQuery += fmt.Sprintf(" AND severity = $%d", argId)
			args = append(args, rule.Severity)
			argId++
		}
		if rule.SourceType != "" {
			countQuery += fmt.Sprintf(" AND source_type = $%d", argId)
			args = append(args, rule.SourceType)
			argId++
		}
		if rule.Keyword != "" {
			countQuery += fmt.Sprintf(" AND payload ILIKE $%d", argId)
			args = append(args, "%"+rule.Keyword+"%")
			argId++
		}

		var count int
		var lastPayload sql.NullString
		err = SQL.DB.QueryRow(countQuery, args...).Scan(&count, &lastPayload)
		if err != nil {
			log.Printf("[ERRO ALERTA] Falha na query de contagem de logs: %v", err)
			continue
		}

		// 4. DISPARAR ALERTA MULTI-CANAL
		if count >= rule.Threshold {
			sample := "Sem payload legível"
			if lastPayload.Valid {
				sample = lastPayload.String
			}

			// Dispara a notificação para TODOS os canais configurados de uma só vez!
			err := multiNotifier.Notify(rule.Name, count, rule.WindowMinutes, sample)
			if err != nil {
				log.Printf("[ERRO ALERTA] %v", err)
			} else {
				// Atualiza a hora na BD com sucesso
				SQL.DB.Exec("UPDATE alert_rules SET last_triggered = NOW() WHERE id = $1", rule.ID)
			}
		}
	}
}

// ServeAlertsView renderiza a página de visualização de regras de alerta
func ServeAlertsView(w http.ResponseWriter, r *http.Request) {
	var rules []models.AlertRule
	rows, err := SQL.DB.Query("SELECT id, enabled, name, severity, source_type, keyword, threshold, window_minutes, last_triggered FROM alert_rules ORDER BY id DESC")
	
	if err == nil && rows != nil {
		defer rows.Close()
		for rows.Next() {
			var ar models.AlertRule
			rows.Scan(&ar.ID, &ar.Enabled, &ar.Name, &ar.Severity, &ar.SourceType, &ar.Keyword, &ar.Threshold, &ar.WindowMinutes, &ar.LastTriggered)
			rules = append(rules, ar)
		}
	}
	
	helpers.RenderTemplate(w, "views/alerts.html", rules)
}

// SaveAlertRule grava uma nova regra de alerta na base de dados
func SaveAlertRule(w http.ResponseWriter, r *http.Request) {
	SQL.DB.Exec("INSERT INTO alert_rules (name, severity, source_type, keyword, threshold, window_minutes) VALUES ($1, $2, $3, $4, $5, $6)", 
		r.FormValue("name"), r.FormValue("severity"), r.FormValue("source_type"), r.FormValue("keyword"), r.FormValue("threshold"), r.FormValue("window"))
	w.Write([]byte(`<div class="p-3 bg-emerald-50 text-emerald-700 rounded-lg border border-emerald-200 text-sm flex items-center font-medium"><i data-lucide="check-circle" class="w-5 h-5 mr-2"></i> Regra base gravada. Motor de avaliação em execução.</div><script>lucide.createIcons();</script>`))
}

// DeleteAlertRule remove uma regra de alerta da base de dados
func DeleteAlertRule(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	SQL.DB.Exec("DELETE FROM alert_rules WHERE id = $1", id)
	ServeAlertsView(w, r)
}