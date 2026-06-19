package main

import (
	"log"
	"syslog-collector/app"
)


func main() {


	defer func() { if r := recover(); r != nil { log.Fatalf("[CRÍTICO] Falha fatal: %v", r) } }()

	rdb := app.InitRedisClient()

	app.InitPolicies(rdb)
	app.InitWorker(rdb)
	app.InitListener()

	log.Println("Collector ativo nas portas 514 (UDP e TCP), a aguardar logs...")
	
}