package settings

import (
	"net/http"
	OS "os"
	"encoding/json"
	"syslog-web/app/helpers"
	"syslog-web/data/SQL"
)


type View interface {
	ServeSettingsView() string
	SaveSettings() error
	ServePoliciesView() string
	SavePolicies() error
}

type Settings struct { 
	Retention int 
	User string 
	Error string
	TgBotUser string
	TgChatID string
	NotifyTelegram bool
	NotifyEmail bool
 }

type SettingsView struct {
	writer http.ResponseWriter
	response *http.Request
}

func NewSettingsView(w http.ResponseWriter, r *http.Request) *SettingsView {
	return &SettingsView{
		writer: w,
		response: r,
	}
}

func (sv *SettingsView) ServeSettingsView(w http.ResponseWriter, r *http.Request) string {
	var settings Settings
	SQL.DB.QueryRow("SELECT retention_days, admin_user, tg_chat_id FROM settings WHERE id = 1").Scan(&settings.Retention, &settings.User, &settings.TgChatID)
	tgBotToken := OS.Getenv("TG_BOT_TOKEN")

	if tgBotToken != "" && tgBotToken != "coloque_aqui_o_token_do_seu_bot" {
		resp, err := http.Get("https://api.telegram.org/bot" + tgBotToken + "/getMe")
		if err == nil {
			defer resp.Body.Close()
			var res struct {
				Ok bool `json:"ok"`
				Result struct {
					Username string `json:"username"`
				} `json:"result"`
			}
			if json.NewDecoder(resp.Body).Decode(&res) == nil && res.Ok {
				settings.TgBotUser = res.Result.Username
			}
	}
	}

	helpers.RenderTemplate(w, "views/settings.html", settings)
	return "views/settings.html"
}		


func (sv *SettingsView) SaveSettings(w http.ResponseWriter, r *http.Request) error {
	retention := r.FormValue("retention")
	user := r.FormValue("username")
	pass := r.FormValue("password")
	tgChat := r.FormValue("tg_chat_id")

	if pass != "" {
		SQL.DB.Exec("UPDATE settings SET retention_days = $1, admin_user = $2, admin_pass = $3, tg_chat_id = $4 WHERE id = 1", retention, user, pass, tgChat)
	} else {
		SQL.DB.Exec("UPDATE settings SET retention_days = $1, admin_user = $2, tg_chat_id = $3 WHERE id = 1", retention, user, tgChat)
	}

	w.Write([]byte(`<div class="p-3 bg-emerald-50 text-emerald-700 rounded-lg text-sm flex items-center font-medium"><i data-lucide="check-circle" class="w-5 h-5 mr-2"></i> Definições atualizadas com sucesso!</div><script>lucide.createIcons();</script>`))
	return nil
}

func (sv *SettingsView) ServeToolsView(w http.ResponseWriter, r *http.Request) string {
	helpers.RenderTemplate(w, "views/tools.html", nil)
	return "views/tools.html"
}

func (sv *SettingsView) ServeDownloadTool(w http.ResponseWriter, r *http.Request) {
	tool := r.URL.Query().Get("tool")
	var c, f string

	switch tool {
	case "firewall":
		c = "@echo off\nstart wf.msc\nexit"; f = "open_firewall.bat"
	case "permissions":
		c = "@echo off\nstart compmgmt.msc\nexit"; f = "manage_permissions.bat"
	case "secpol":
		c = "@echo off\nstart secpol.msc\nexit"; f = "security_policies.bat"
	default:
		http.Error(w, "Ferramenta não encontrada", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+f)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write([]byte(c))
}

func (sv *SettingsView) ServePoliciesView() string {
	// Lógica para servir a visualização de políticas
	return "views/policies.html"
}

func (sv *SettingsView) SavePolicies() error {
	// Lógica para salvar as políticas
	return nil
}