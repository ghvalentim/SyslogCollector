package main

import (

	"syslog-web/app"
	"syslog-web/database"
	_ "github.com/lib/pq"
)


func main() {


	database.InitData()
	app.InitServices()
	app.InitRoutes()
	app.InitAlerts()


}





