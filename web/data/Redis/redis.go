package Redis

import (
	"context"
	"log"
	"os"
	"github.com/redis/go-redis/v9"
	_ "github.com/lib/pq"
	
	
)


var (
	Rdb *redis.Client
	Ctx = context.Background()
)

// --- REDIS ---
// InitRedis inicializa o cliente Redis e verifica a conectividade.
func InitRedis() {
	Rdb = redis.NewClient(&redis.Options{Addr: os.Getenv("REDIS_URL"), DB: 0})
	if _, err := Rdb.Ping(Ctx).Result(); err != nil {
		log.Fatalf("Erro Redis: %v", err)
	}
}

