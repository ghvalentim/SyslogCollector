package app

import (
	"net/http"
	_ "github.com/redis/go-redis/v9"
	_ "github.com/lib/pq"
)

// --- DEFINIÇÕES E FERRAMENTAS ---
func ServeSettingsView(w http.ResponseWriter, r *http.Request) {
	var s Settings
	DB.QueryRow("SELECT retention_days, admin_user, tg_bot_token, tg_chat_id FROM settings WHERE id = 1").Scan(&s.Retention, &s.User, &s.TgToken, &s.TgChatID)
	RenderTemplate(w, "templates/settings.html", s)
}

func SaveSettings(w http.ResponseWriter, r *http.Request) {
	retention := r.FormValue("retention")
	user := r.FormValue("username")
	pass := r.FormValue("password")
	tgToken := r.FormValue("tg_token")
	tgChat := r.FormValue("tg_chat_id")

	if pass != "" {
		DB.Exec("UPDATE settings SET retention_days = $1, admin_user = $2, admin_pass = $3, tg_bot_token = $4, tg_chat_id = $5 WHERE id = 1", retention, user, pass, tgToken, tgChat)
	} else {
		DB.Exec("UPDATE settings SET retention_days = $1, admin_user = $2, tg_bot_token = $3, tg_chat_id = $4 WHERE id = 1", retention, user, tgToken, tgChat)
	}

	w.Write([]byte(`<div class="p-3 bg-emerald-50 text-emerald-700 rounded-lg text-sm flex items-center font-medium"><i data-lucide="check-circle" class="w-5 h-5 mr-2"></i> Definições atualizadas com sucesso!</div><script>lucide.createIcons();</script>`))
}

func ServeToolsView(w http.ResponseWriter, r *http.Request) { RenderTemplate(w, "templates/tools.html", nil) }

func DownloadTool(w http.ResponseWriter, r *http.Request) {
	var c, f string
	switch r.URL.Query().Get("action") {
	case "firewall": c = "@echo off\necho A abrir a Firewall...\nstart wf.msc\nexit"; f = "abrir_firewall.bat"
	case "compmgmt": c = "@echo off\necho A abrir Permissoes...\nstart compmgmt.msc\nexit"; f = "gerir_permissoes.bat"
	case "secpol": c = "@echo off\necho A abrir Secpol...\nstart secpol.msc\nexit"; f = "politicas_seguranca.bat"
	default: http.Error(w, "Inválido", 400); return
	}
	w.Header().Set("Content-Disposition", "attachment; filename="+f); w.Header().Set("Content-Type", "application/bat")
	w.Write([]byte(c))
}