package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"

	pb "grpc/proto"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/segmentio/kafka-go"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedWeatherTweetServiceServer
	kafkaBroker string
	kafkaTopic  string
	rabbitURL   string
	target      string // "both" | "kafka" | "rabbit" | "random"
}

type Clima struct {
	Municipio   string `json:"municipio"`
	Temperatura int    `json:"temperatura"`
	Humedad     int    `json:"humedad"`
	Clima       string `json:"clima"`
}

func main() {
	// Defaults “pensando en K8s”; para local sobreescribe con export (ver pasos)
	kafkaBroker := env("KAFKA_BROKER", "kafka:9092")
	kafkaTopic := env("KAFKA_TOPIC", "clima")
	rabbitURL := env("RABBITMQ_URL", "amqp://guest:guest@rabbitmq:5672/")
	target := strings.ToLower(env("WRITER_TARGET", "both")) // both|kafka|rabbit|random

	// semilla para random si usas WRITER_TARGET=random
	rand.Seed(time.Now().UnixNano())

	s := &server{
		kafkaBroker: kafkaBroker,
		kafkaTopic:  kafkaTopic,
		rabbitURL:   rabbitURL,
		target:      target,
	}

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("❌ Error escuchando: %v", err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterWeatherTweetServiceServer(grpcServer, s)

	fmt.Printf("Go gRPC server escuchando en :50051  (kafka=%s topic=%s  rabbit=%s  target=%s)\n",
		kafkaBroker, kafkaTopic, rabbitURL, target)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("❌ Error en Serve: %v", err)
	}
}

func (s *server) SendTweet(ctx context.Context, req *pb.WeatherTweetRequest) (*pb.WeatherTweetResponse, error) {
	clima := Clima{
		Municipio:   muniToStr(req.GetMunicipality()),
		Temperatura: int(req.GetTemperature()),
		Humedad:     int(req.GetHumidity()),
		Clima:       weatherToStr(req.GetWeather()),
	}
	payload, _ := json.Marshal(clima)

	fmt.Printf("📦 Recibido por gRPC: %+v → JSON=%s\n", req, string(payload))

	var err error
	switch s.target {
	case "kafka":
		err = publishKafka(s.kafkaBroker, s.kafkaTopic, payload)
	case "rabbit":
		err = publishRabbit(s.rabbitURL, payload)
	case "random":
		if rand.Intn(2) == 0 {
			err = publishKafka(s.kafkaBroker, s.kafkaTopic, payload)
		} else {
			err = publishRabbit(s.rabbitURL, payload)
		}
	default: // both
		if e := publishKafka(s.kafkaBroker, s.kafkaTopic, payload); e != nil {
			err = e
		}
		if e := publishRabbit(s.rabbitURL, payload); e != nil {
			err = e
		}
	}

	if err != nil {
		return nil, fmt.Errorf("writer error: %w", err)
	}
	return &pb.WeatherTweetResponse{Status: "Tweet recibido y publicado ✅"}, nil
}

func publishKafka(broker, topic string, msg []byte) error {
	w := &kafka.Writer{
		Addr:         kafka.TCP(broker),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireAll,
		Async:        false,
	}
	defer w.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return w.WriteMessages(ctx, kafka.Message{Value: msg})
}

func publishRabbit(url string, msg []byte) error {
	conn, err := amqp.Dial(url)
	if err != nil {
		return err
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()
	q, err := ch.QueueDeclare("clima", false, false, false, false, nil)
	if err != nil {
		return err
	}
	return ch.Publish("", q.Name, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        msg,
	})
}

func muniToStr(m pb.Municipalities) string {
	switch m {
	case pb.Municipalities_mixco:
		return "Mixco"
	case pb.Municipalities_guatemala:
		return "Guatemala"
	case pb.Municipalities_amatitlan:
		return "Amatitlán"
	case pb.Municipalities_chinautla:
		return "Chinautla"
	default:
		return "Desconocido"
	}
}

func weatherToStr(w pb.Weathers) string {
	switch w {
	case pb.Weathers_sunny:
		return "Soleado"
	case pb.Weathers_cloudy:
		return "Nublado"
	case pb.Weathers_rainy:
		return "Lluvioso"
	case pb.Weathers_foggy:
		return "Ventoso"
	default:
		return "Desconocido"
	}
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
