package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

type Clima struct {
	Municipio   string `json:"municipio"`
	Temperatura int    `json:"temperatura"`
	Humedad     int    `json:"humedad"`
	Clima       string `json:"clima"`
}

func main() {
	// 1) RabbitMQ
	url := env("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	conn, err := amqp.Dial(url)
	if err != nil {
		log.Fatal("❌ Rabbit Dial:", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal("❌ Rabbit Channel:", err)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(env("RABBIT_QUEUE", "clima"), false, false, false, false, nil)
	if err != nil {
		log.Fatal("❌ QueueDeclare:", err)
	}

	msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		log.Fatal("❌ Consume:", err)
	}

	// 2) Valkey
	rdb := newValkeyClient(env("VALKEY_SERVICE_URL", "localhost:6379"))

	fmt.Println("📥 Rabbit consumer listo. Escribiendo en Valkey…")
	for d := range msgs {
		var c Clima
		if err := json.Unmarshal(d.Body, &c); err != nil {
			log.Println("❌ JSON inválido:", err, "payload=", string(d.Body))
			continue
		}
		if err := writeToValkey(rdb, c); err != nil {
			log.Println("❌ Valkey write:", err)
			continue
		}
		log.Printf("✅ [Rabbit→Valkey] %s", c.Municipio)
	}
}

func writeToValkey(rdb *redis.Client, c Clima) error {
	slug := slugify(c.Municipio)
	now := time.Now()
	// Estado actual
	key := fmt.Sprintf("municipality:%s", slug)
	if err := rdb.HSet(ctx, key,
		"name", c.Municipio,
		"temperature", c.Temperatura,
		"humidity", c.Humedad,
		"weather", c.Clima,
		"last_update", now.Format("2006-01-02 15:04:05"),
	).Err(); err != nil {
		return err
	}
	_ = rdb.Expire(ctx, key, time.Hour)

	// Histórico en STREAM
	stream := fmt.Sprintf("clima:%s", slug)
	return rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: map[string]any{
			"ts":          now.UnixMilli(),
			"temperature": c.Temperatura,
			"humidity":    c.Humedad,
			"weather":     c.Clima,
		},
	}).Err()
}

func newValkeyClient(addr string) *redis.Client {
	if strings.HasPrefix(addr, "redis://") {
		opt, err := redis.ParseURL(addr)
		if err != nil {
			log.Fatal("URL inválida en VALKEY_SERVICE_URL:", err)
		}
		return redis.NewClient(opt)
	}
	return redis.NewClient(&redis.Options{Addr: addr})
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	repl := map[string]string{"á": "a", "é": "e", "í": "i", "ó": "o", "ú": "u", "ñ": "n"}
	for k, v := range repl {
		s = strings.ReplaceAll(s, k, v)
	}
	return s
}
func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
