package redis

import (
	"os"
	"github.com/redis/go-redis/v9"
	"syslog-collector/settings"
)

var ctx = settings.GetContext()

func InitRedisClient() *redis.Client {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" { redisURL = "redis:6379" }
	rdb := redis.NewClient(&redis.Options{Addr: redisURL, DB: 0})
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		panic("Erro Redis: " + err.Error())
	}
	return rdb
}

/* Inicialização do cliente Redis, com configuração via variável de ambiente.
O cliente é usado para armazenar logs processados, manter estatísticas, e gerenciar políticas de filtragem.
Pode ser expandido para suportar autenticação, TLS, ou outros bancos de dados. */