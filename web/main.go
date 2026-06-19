package main

import (

	"log"
	"syslog-web/app"
	_ "github.com/lib/pq"
)

// --- ESTRUTURAS DE DADOS ---


func main() {
	defer func() { if r := recover(); r != nil { log.Fatalf("[CRÍTICO] Falha fatal: %v", r) } }()

	app.InitData()

	app.InitServices()

	app.InitRoutes()



}



// --- LOGIN ---


// --- MÓDULO DE LOGS ---








