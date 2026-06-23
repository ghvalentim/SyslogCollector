package main

import (

	"syslog-web/app"
	"syslog-web/database"
	"syslog-web/api/telegram"
	_ "github.com/lib/pq"
)


func main() {


	database.InitData()
	app.InitServices()
	app.InitRoutes()
	telegram.InitBot()
	app.InitAlerts()


}





