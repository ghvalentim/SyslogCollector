package main

import (
	"syslog-web/app/logs"
	"syslog-web/app/routes"
	"syslog-web/data/SQL"
	"syslog-web/data/Redis"
	"syslog-web/app/tools"
	"net/http"
	"log"
	_ "github.com/lib/pq"
)



func main() {

	Redis.InitRedis()
	SQL.InitSQL()
	logs.InitWorkers()
	tools.InitAlerts()
	routes.InitRoutes()

	// Inicia o servidor HTTP na porta 8080
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Erro ao iniciar o servidor: %v", err)
	}


}





