package settings

import (
	"net/http"
	_ "github.com/redis/go-redis/v9"
	_ "github.com/lib/pq"
)

// Executa settings-view.go

func LoadSettingsView() http.HandlerFunc {
	view := ServeSettingsView()
	return view
}

