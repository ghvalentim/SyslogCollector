package main

import (

	"syslog-web/app"
	"syslog-web/api"
	"syslog-web/database"
	_ "github.com/lib/pq"
)


func main() {


	database.InitData()
	app.InitServices()
	app.InitRoutes()
	api.InitTelegramBot()
	app.InitAlerts()


}





