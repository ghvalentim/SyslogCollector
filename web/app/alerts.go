package app

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	notifier "syslog-web/api/notifier"
	database "syslog-web/database"
	model "syslog-web/models"
	"time"
)

func InitAlerts() {
	go StartAlertEngine()
}

// --- ALERTAS ---

// StartAlertEngine inicia a verificação periódica de regras (Corre a cada 30 segundos)
func StartAlertEngine() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		evaluateAlertRules()
	}
}

func evaluateAlertRules() {
	if database.DB == nil {
		return
	}

	// 1. Ir buscar as configurações (Token seguro do .env e Chat ID da BD)
	tgToken := os.Getenv("TG_BOT_TOKEN")
	var tgChat string

	err := database.DB.QueryRow("SELECT tg_chat_id FROM settings WHERE id = 1").Scan(&tgChat)
	if err != nil || tgToken == "" || tgChat == "" || tgToken == "coloque_aqui_o_seu_token_do_botfather" {
		return // Não há notificações configuradas completamente, aborta a avaliação
	}

	notifierClient := notifier.NewTelegramNotifier(tgToken, tgChat)

	// 2. Ir buscar as regras ATIVAS
	rows, err := database.DB.Query("SELECT id, name, severity, source_type, keyword, threshold, window_minutes, last_triggered FROM alert_rules WHERE enabled = true")
	if err != nil {
		log.Printf("[ERRO ALERTA] Falha a ler regras da base de dados: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var rule model.AlertRule
		if err := rows.Scan(&rule.ID, &rule.Name, &rule.Severity, &rule.SourceType, &rule.Keyword, &rule.Threshold, &rule.WindowMinutes, &rule.LastTriggered); err != nil {
			continue
		}

		// Prevenir spam: Só permite disparar a mesma regra após o tempo da janela ter passado novamente
		if rule.LastTriggered.Valid && time.Since(rule.LastTriggered.Time) < time.Duration(rule.WindowMinutes)*time.Minute {
			continue
		}

		// 3. Contar ocorrências nos últimos X minutos (Query Dinâmica)
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
		err = database.DB.QueryRow(countQuery, args...).Scan(&count, &lastPayload)
		if err != nil {
			log.Printf("[ERRO ALERTA] Falha na query de contagem de logs: %v", err)
			continue
		}

		// 4. Se passou o limite de ocorrências, DISPARAR ALERTA!
		if count >= rule.Threshold {
			sample := "Sem payload legível"
			if lastPayload.Valid {
				sample = lastPayload.String
			}

			// Dispara a notificação. Qualquer erro de API/JSON volta empacotado aqui (%w)
			if notifierClient == nil {
				continue
			}
			err := notifierClient.Notify(rule.Name, count, rule.WindowMinutes, sample)
			if err != nil {
				log.Printf("[ERRO ALERTA] %v", err) // Agora o log de erro será detalhado e exato
			} else {
				// Atualiza a hora do último disparo na BD com sucesso para evitar spam
				database.DB.Exec("UPDATE alert_rules SET last_triggered = NOW() WHERE id = $1", rule.ID)
			}
		}
	}
}

func ServeAlertsView(w http.ResponseWriter, r *http.Request) {
	var rules []model.AlertRule
	rows, err := database.DB.Query("SELECT id, enabled, name, severity, source_type, keyword, threshold, window_minutes, last_triggered FROM alert_rules ORDER BY id DESC")

	if err == nil && rows != nil {
		defer rows.Close()
		for rows.Next() {
			var ar model.AlertRule
			rows.Scan(&ar.ID, &ar.Enabled, &ar.Name, &ar.Severity, &ar.SourceType, &ar.Keyword, &ar.Threshold, &ar.WindowMinutes, &ar.LastTriggered)
			rules = append(rules, ar)
		}
	}

	RenderTemplate(w, "templates/alerts.html", rules)
}

func SaveAlertRule(w http.ResponseWriter, r *http.Request) {
	database.DB.Exec("INSERT INTO alert_rules (name, severity, source_type, keyword, threshold, window_minutes) VALUES ($1, $2, $3, $4, $5, $6)",
		r.FormValue("name"), r.FormValue("severity"), r.FormValue("source_type"), r.FormValue("keyword"), r.FormValue("threshold"), r.FormValue("window"))
	w.Write([]byte(`<div class="p-3 bg-emerald-50 text-emerald-700 rounded-lg border border-emerald-200 text-sm flex items-center font-medium"><i data-lucide="check-circle" class="w-5 h-5 mr-2"></i> Regra base gravada. Motor de avaliação em execução.</div><script>lucide.createIcons();</script>`))
}

func DeleteAlertRule(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	database.DB.Exec("DELETE FROM alert_rules WHERE id = $1", id)

	// Retorna a vista atualizada (HTMX substitui a tabela automaticamente)
	ServeAlertsView(w, r)
}
