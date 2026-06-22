package main

import (

	"log"
	"syslog-web/app"
	_ "github.com/lib/pq"
)


func main() {
	defer func() { if r := recover(); r != nil { log.Fatalf("[CRÍTICO] Falha fatal: %v", r) } }()

	app.InitData()
	app.InitServices()
	app.InitRoutes()
	app.InitTelegramBot()
	app.InitAlerts()

}





