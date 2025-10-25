// main.go (smoke test)
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

func main() {
	addr := os.Getenv("VALKEY_SERVICE_URL")
	if addr == "" {
		addr = "localhost:6379"
	}

	var rdb *redis.Client
	if len(addr) >= 8 && addr[:8] == "redis://" {
		opt, err := redis.ParseURL(addr)
		if err != nil {
			log.Fatal(err)
		}
		rdb = redis.NewClient(opt)
	} else {
		rdb = redis.NewClient(&redis.Options{Addr: addr})
	}

	if pong, err := rdb.Ping(ctx).Result(); err != nil {
		log.Fatal("Error conectando a Valkey:", err)
	} else {
		fmt.Println("Conexión exitosa:", pong)
	}
}
