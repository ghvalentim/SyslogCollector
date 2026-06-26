package main

import (
	"log"
	"syslog-collector/redis"
	"syslog-collector/settings"
	"syslog-collector/tools"
)


func main() {


	defer func() { if r := recover(); r != nil { log.Fatalf("[CRÍTICO] Falha fatal: %v", r) } }()

	rdb := redis.InitRedisClient()

	settings.InitPolicies(rdb)
	tools.InitWorker(rdb)
	tools.InitListener()

	log.Println("Collector ativo nas portas 514 (UDP e TCP), a aguardar logs...")
	
}