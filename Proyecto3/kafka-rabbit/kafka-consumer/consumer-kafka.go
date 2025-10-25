package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
)

var ctx = context.Background()

type Clima struct {
	Municipio   string `json:"municipio"`
	Temperatura int    `json:"temperatura"`
	Humedad     int    `json:"humedad"`
	Clima       string `json:"clima"`
}

func main() {
	brokers := env("KAFKA_BROKER", "localhost:9092")
	topic := env("KAFKA_TOPIC", "clima")
	group := env("KAFKA_GROUP", "clima-consumer-group") // para depurar puedes pasar un grupo nuevo

	log.Printf("Kafka consumer: brokers=%s topic=%s group=%s", brokers, topic, group)

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{brokers},
		Topic:   topic,
		GroupID: group,
	})

	// Valkey
	rdb := newValkeyClient(env("VALKEY_SERVICE_URL", "localhost:6379"))

	fmt.Println("📥 Kafka consumer listo. Escribiendo en Valkey…")
	for {
		m, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Println("❌ Kafka read:", err)
			continue
		}
		var c Clima
		if err := json.Unmarshal(m.Value, &c); err != nil {
			log.Println("❌ JSON inválido:", err, "payload=", string(m.Value))
			continue
		}
		if err := writeToValkey(rdb, c); err != nil {
			log.Println("❌ Valkey write:", err)
			continue
		}
		fmt.Println("kafka a valkey, recibido aaa")
		log.Printf("✅ [Kafka→Valkey] %s", c.Municipio)
	}
}

func writeToValkey(rdb *redis.Client, c Clima) error {
	slug := slugify(c.Municipio)
	now := time.Now()
	// Estado actual (HASH)
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

	// Histórico (STREAM)
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
