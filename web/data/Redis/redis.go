package Redis

import (
	"log"
	"os"
	"context"
	"github.com/redis/go-redis/v9"
	_ "github.com/lib/pq"
		
)

var (
	Rdb = getClient()
	Ctx = getContext()
)

// --- REDIS ---
// InitRedis inicializa o cliente Redis e verifica a conectividade.
func InitRedis() {
	if _, err := Rdb.Ping(Ctx).Result(); err != nil {
		log.Fatalf("Erro Redis: %v", err)
	} else {
		log.Println("Conexão com o Redis estabelecida com sucesso.")
	}
}

func getClient() *redis.Client {
	Rdb := redis.NewClient(&redis.Options{Addr: os.Getenv("REDIS_URL"), DB: 0})
	return Rdb
}

func getContext() context.Context {
	Ctx := context.Background()
	return Ctx
}

