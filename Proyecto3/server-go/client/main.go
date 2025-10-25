package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	pb "grpc/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type In struct {
	// Recibimos strings para mapear a los enums del .proto (solo si usas el endpoint HTTP /tweet)
	Municipality string `json:"municipality"` // "mixco","guatemala","amatitlan","chinautla"
	Temperature  int    `json:"temperature"`
	Humidity     int    `json:"humidity"`
	Weather      string `json:"weather"` // "sunny","cloudy","rainy","foggy"
}

func main() {
	// 1) Resolver dirección del server-go: GRPC_SERVER_URL / WRITER_ADDR / fallback
	raw := firstNonEmpty(
		os.Getenv("GRPC_SERVER_URL"), // p.ej. "http://server-go:50051" (estilo Rust)
		os.Getenv("WRITER_ADDR"),     // p.ej. "server-go:50051" (estilo Go)
		"localhost:50051",
	)
	grpcAddr := normalizeGrpcAddr(raw) // quita http:// o https:// si viene

	// 2) Levantar gRPC proxy del client-go para que Rust se conecte aquí
	proxyListen := env("CLIENT_GRPC_LISTEN", ":50052")
	go func() {
		if err := runGRPCProxy(proxyListen, grpcAddr); err != nil {
			log.Fatalf("grpc proxy error: %v", err)
		}
	}()

	// 3) Conexión gRPC saliente (reutilizada) hacia server-go (para el endpoint HTTP /tweet)
	conn, err := grpc.Dial(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	check(err)
	defer conn.Close()
	client := pb.NewWeatherTweetServiceClient(conn)

	// 4) HTTP (opcional) - healthcheck
	http.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Estamos vivos, desde el client-go"))
	})

	// 5) HTTP (opcional) - endpoint REST que reenvía a gRPC server-go
	http.HandleFunc("/tweet", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "use POST", http.StatusMethodNotAllowed)
			return
		}

		var in In
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}

		req := &pb.WeatherTweetRequest{
			Municipality: toMunicipalityEnum(in.Municipality),
			Temperature:  int32(in.Temperature),
			Humidity:     int32(in.Humidity),
			Weather:      toWeatherEnum(in.Weather),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		if _, err := client.SendTweet(ctx, req); err != nil {
			log.Printf("gRPC error: %v", err)
			http.Error(w, "grpc error", http.StatusBadGateway)
			return
		}

		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	port := env("PORT", "8081")
	log.Printf("Client-Go HTTP :%s | gRPC proxy %s → upstream=%s", port, proxyListen, grpcAddr)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func toMunicipalityEnum(s string) pb.Municipalities {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "mixco":
		return pb.Municipalities_mixco
	case "guatemala":
		return pb.Municipalities_guatemala
	case "amatitlan":
		return pb.Municipalities_amatitlan
	case "chinautla":
		return pb.Municipalities_chinautla
	default:
		return pb.Municipalities_unknown_municipalities
	}
}

func toWeatherEnum(s string) pb.Weathers {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "sunny":
		return pb.Weathers_sunny
	case "cloudy":
		return pb.Weathers_cloudy
	case "rainy":
		return pb.Weathers_rainy
	case "foggy":
		return pb.Weathers_foggy
	default:
		return pb.Weathers_unknown_weather
	}
}

// --- gRPC proxy (el client también expone gRPC y reenvía al server-go) ---

type proxyServer struct {
	pb.UnimplementedWeatherTweetServiceServer
	upstream string // host:port del server-go gRPC
}

func (p *proxyServer) SendTweet(ctx context.Context, req *pb.WeatherTweetRequest) (*pb.WeatherTweetResponse, error) {
	log.Printf("proxy: → reenviando a %s  (municipality=%v temp=%d hum=%d weather=%v)",
		p.upstream, req.GetMunicipality(), req.GetTemperature(), req.GetHumidity(), req.GetWeather())

	conn, err := grpc.Dial(p.upstream, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial upstream: %w", err)
	}
	defer conn.Close()

	c := pb.NewWeatherTweetServiceClient(conn)
	fmt.Printf("holaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	return c.SendTweet(ctx, req)
}

func runGRPCProxy(listenAddr, upstream string) error {
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}
	s := grpc.NewServer()
	pb.RegisterWeatherTweetServiceServer(s, &proxyServer{upstream: upstream})
	log.Printf("client-go gRPC proxy escuchando en %s → upstream=%s", listenAddr, upstream)
	return s.Serve(lis)
}

// --- utils ---

func normalizeGrpcAddr(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "http://")
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "dns:///")
	return s
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
