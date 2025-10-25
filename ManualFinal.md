# Manual Técnico: Proyecto 3 - Tweets del Clima

## Locust
Esta herramienta se encarga de generar trafico, para hacer uso de ella hicimos se creo un directorio llamado `locus`, dentro de este se ejecuto el siguiente comando:

```bash
pip install locust
```

Una vez ejecutado dicho comando tendremos locust para poder realiza pruebas, en mi caso, agregue el siguiente codigo en un archivo llamado `locust.py`.

```py
from locust import HttpUser, TaskSet, task, between
import random
import json

class MyTasks(TaskSet):
    
    @task(1)
    def engineering(self):
        # Lista de municipios
        municipios = ["mixco", "guatemala", "amatitlan", "chinautla"]

        # Lista de condiciones climáticas
        climas = ["sunny", "cloudy", "rainy", "foggy"]
    
        # Datos meteorológicos (simulados)
        weather_data = {
            "municipality": random.randint(1,4),  # Municipio aleatorio
            "temperature": random.randint(18, 28),  # Temperatura aleatoria entre 18 y 28
            "humidity": random.randint(40, 80),  # Humedad aleatoria entre 40 y 80
            "weather": random.randint(1,4)  # Clima aleatorio
        }
        
        # Enviar el JSON como POST a la ruta /clima
        headers = {'Content-Type': 'application/json'}
        self.client.post("/clima", json=weather_data, headers=headers)

class WebsiteUser(HttpUser):
    host = "http://136.117.88.73"  # Cambia esto a la IP externa de tu API Rust en GCP
    tasks = [MyTasks]
    wait_time = between(1, 5)  # Tiempo de espera entre tareas (de 1 a 5 segundos)

```

Esto nos permitira usar locust desde el puerto `8089`, esto se puede visualizar desde consola.

## api-rust
La API en Rust es un servicio que actúa como intermediario entre el generador de carga (Locust) y los servicios de procesamiento de mensajes (Kafka, RabbitMQ). Su función principal es recibir las peticiones HTTP que contienen datos climáticos, procesarlos y luego enviarlos a los servicios de Go que se encargarán de distribuirlos a los brokers de mensajes (Kafka o RabbitMQ). Aquí te explico cómo funciona todo el proceso:

1. El Rol de la API en Rust

La API REST en Rust está diseñada para recibir datos en formato JSON mediante el endpoint /clima. Estos datos contienen información sobre el municipio, la temperatura, la humedad y las condiciones climáticas (como "soleado", "nublado", "lluvioso", etc.).

La API en Rust hace lo siguiente:

Recibe peticiones HTTP POST: Recibe las peticiones enviadas por Locust a través de una solicitud HTTP POST. Cada solicitud contiene un JSON con los datos climáticos.

Valida y procesa los datos: La API deserializa el JSON recibido, mapea los datos a las estructuras correspondientes (usando Rust y el framework serde).

Conecta con los servicios Go: La API en Rust utiliza gRPC para comunicarse con los servicios Go. Dependiendo de los parámetros configurados, la API envía los datos hacia Kafka o RabbitMQ.

Devuelve una respuesta HTTP: Después de enviar los datos a Kafka o RabbitMQ, la API en Rust devuelve una respuesta al cliente con el estado de la operación.

2. Flujo de Peticiones

Locust envía peticiones a la API de Rust:

Locust genera tráfico de peticiones HTTP hacia el endpoint /clima de la API en Rust.

Las peticiones contienen información sobre el clima, por ejemplo:

```json
{
    "municipality": "guatemala",
    "temperature": 25,
    "humidity": 60,
    "weather": "sunny"
}
```

La API en Rust recibe las peticiones:

La API deserializa el JSON y extrae los valores de cada campo.

Convierte el municipio, la temperatura, la humedad y el clima en un formato adecuado para enviarlo a los servicios Go.

La API en Rust se comunica con los servicios Go:

Después de procesar los datos, la API en Rust hace una llamada gRPC a los servicios Go. Esta llamada envía los datos a Kafka o RabbitMQ, según la configuración.

gRPC es un sistema de comunicación de alto rendimiento basado en HTTP/2, ideal para servicios distribuidos como el nuestro.

Kafka o RabbitMQ procesan los mensajes:

Los servicios Go actúan como "writers" que publican los mensajes en los brokers de mensajes, Kafka o RabbitMQ.

Kafka o RabbitMQ se encargan de manejar la cola de mensajes, asegurando que los mensajes sean procesados de manera eficiente y en el orden adecuado.

El cliente recibe la respuesta:

La API en Rust responde al cliente que hizo la petición con un JSON que contiene el estado de la operación. Por ejemplo:

```go
{
    "status": "Tweet recibido y publicado ✅"
}
```
3. Arquitectura en GCP

Cuando se despliega en Google Cloud Platform (GCP), la API en Rust se ejecuta dentro de un cluster de Kubernetes (GKE). El flujo de datos en la nube es el siguiente:

Locust genera tráfico:

Locust está corriendo en el entorno local (o en otra parte de GCP), y realiza solicitudes HTTP a la API en Rust usando la IP externa del LoadBalancer de Kubernetes.

NGINX Ingress Controller:

El NGINX Ingress Controller en el cluster de Kubernetes recibe las solicitudes HTTP y las enruta al servicio adecuado. En este caso, las solicitudes son enviadas a la API en Rust.

La API en Rust recibe las solicitudes:

La API en Rust es un Pod de Kubernetes en el namespace clima-app que escucha las solicitudes HTTP en el puerto 8080.

La API procesa los datos y se comunica con los servicios Go para enviar los datos a Kafka o RabbitMQ.

Comunicaciones entre Kubernetes y GKE:

La comunicación entre los diferentes Pods (API en Rust, servicios Go, Kafka, RabbitMQ) se realiza internamente a través de los servicios de Kubernetes (que se exponen en puertos internos y se comunican usando ClusterIP).

El tráfico hacia los servicios de Go se realiza mediante gRPC utilizando las direcciones de servicio de Kubernetes (por ejemplo, server-go-service:50051).

4. Estructura del Proyecto

La API en Rust está configurada dentro de un contenedor Docker, y su Dockerfile se ve como sigue:
```bash
FROM rust:alpine AS builder

# Crear directorio de trabajo
WORKDIR /app

RUN apk add --no-cache protobuf gcc musl-dev

# Copiar archivos de configuración
COPY . .

# Compilar la aplicación en modo release
RUN cargo install --path .

# Imagen final más pequeña
FROM alpine

# Crear directorio de trabajo
WORKDIR /app

# Copiar el binario compilado desde el builder
COPY --from=builder --chown=app_user:app_user /app/target/release/api-rust ./api-rust

# Exponer el puerto en el que corre la aplicación
EXPOSE 8080

# Agregar variable de entorno para la URL del servidor gRPC
ENV GRPC_SERVER_URL=http://server-go-service:50052

RUN adduser -D app_user
USER app_user

# Comando para ejecutar la aplicación
CMD ["./api-rust"]
```

Este Dockerfile crea la imagen de la API en Rust, que se utiliza para crear el Pod de Kubernetes.

5. Despliegue de la API en GKE

El despliegue de la API en Rust se hace mediante un archivo YAML, que se aplica con kubectl:

```bash
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-rust
  namespace: clima-app
  labels: { app: api-rust }
spec:
  replicas: 1
  selector: { matchLabels: { app: api-rust } }
  template:
    metadata: { labels: { app: api-rust } }
    spec:
      containers:
      - name: api-rust
        image: nonalcoholic-dixie-freshly.ngrok-free.dev/api-rust:v1           # <-- cambia esto
        imagePullPolicy: Always
        env:
        - name: GRPC_SERVER_URL          # Rust → client-go proxy gRPC
          value: "http://client-go-service:50052"
        ports:
        - name: http
          containerPort: 8080
        readinessProbe:
          httpGet: { path: /, port: 8080 }
          initialDelaySeconds: 5
          periodSeconds: 5
        livenessProbe:
          httpGet: { path: /, port: 8080 }
          initialDelaySeconds: 10
          periodSeconds: 10
        resources:
          requests: { cpu: "100m", memory: "128Mi" }
          limits:   { cpu: "500m", memory: "512Mi" }
---
apiVersion: v1
kind: Service
metadata:
  name: api-rust-service
  namespace: clima-app
  labels: { app: api-rust }
spec:
  # Para pruebas rápidas lo exponemos; si prefieres Ingress, cambia a ClusterIP y crea el Ingress
  type: LoadBalancer
  selector: { app: api-rust }
  ports:
  - name: http
    port: 80
    targetPort: 8080

```
Este archivo describe el despliegue de la API en Kubernetes y expone el puerto 8080.
## server
El server es el componente que expone una API gRPC para recibir peticiones de los diferentes clientes (como el cliente Go o la API Rust). Su tarea es recibir los datos climáticos enviados por los clientes y procesarlos, enviándolos luego a uno de los Message Brokers (Kafka o RabbitMQ).

### Flujo de la comunicación:
#### Recepción de datos:

El server escucha las peticiones en el puerto 50051 (a través de gRPC).

La API que se conecta con este server, como la API en Rust, utiliza el protocolo gRPC para enviar solicitudes.

#### Procesamiento de mensajes:

Los datos recibidos por el server se procesan (el JSON se convierte en objetos internos).

En base al valor de la variable de entorno WRITER_TARGET, el servidor decide si los mensajes se envían a Kafka, RabbitMQ, o a ambos.

Si la variable WRITER_TARGET está configurada como random, el servidor decide aleatoriamente entre Kafka y RabbitMQ.

`Envío a Kafka o RabbitMQ:`

Después de procesar los datos, el server los envía al broker correspondiente.

Si los datos deben enviarse a Kafka, el server usa la librería kafka-go para escribir los mensajes.

Si los datos deben enviarse a RabbitMQ, el server usa la librería amqp091-go para enviar los mensajes.

`Respuesta`:

Después de enviar los datos a Kafka o RabbitMQ, el server responde al cliente que hizo la solicitud.

La respuesta es un JSON que contiene el estado de la operación, por ejemplo: "status": "Tweet recibido y publicado ✅".

Código del server (Go)

Aquí hay un ejemplo del código de tu server en Go:
```go
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
	kafkaBroker := env("KAFKA_BROKER", "kafka:9092")
	kafkaTopic := env("KAFKA_TOPIC", "clima")
	rabbitURL := env("RABBITMQ_URL", "amqp://guest:guest@rabbitmq:5672/")
	target := strings.ToLower(env("WRITER_TARGET", "both")) // both|kafka|rabbit|random

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
```

`Resumen del servidor`:

`gRPC`: Se usa para recibir solicitudes desde el cliente (por ejemplo, API en Rust o client-go) y enviar respuestas.

`Kafka / RabbitMQ`: Los mensajes de clima se envían a Kafka o RabbitMQ según lo que esté configurado en el parámetro WRITER_TARGET. Si es random, el servidor elige entre los dos.

## client
El Client Go es el componente que actúa como intermediario entre el cliente HTTP (como Locust) y el server-go. En su configuración, el cliente puede ser un servidor que también expone un endpoint HTTP para recibir las peticiones de Locust y las convierte en llamadas gRPC a server-go.

### Flujo de la comunicación del Client:

#### Petición HTTP (de Locust):

Locust genera solicitudes HTTP a los endpoints definidos en el client-go.

El cliente convierte estas solicitudes en llamadas gRPC y las envía al server-go.

### Envío de datos a Server:

El cliente empaqueta los datos recibidos en el cuerpo de la solicitud en un formato que el server-go espera (en este caso, como una solicitud de gRPC).

#### gRPC Proxy:

El cliente también actúa como un proxy gRPC, reenviando la solicitud de gRPC recibida desde Locust a server-go.

### Código del Client:

```go
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
	Municipality string `json:"municipality"`
	Temperature  int    `json:"temperature"`
	Humidity     int    `json:"humidity"`
	Weather      string `json:"weather"`
}

func main() {
	raw := firstNonEmpty(
		os.Getenv("GRPC_SERVER_URL"),
		os.Getenv("WRITER_ADDR"),
		"localhost:50051",
	)
	grpcAddr := normalizeGrpcAddr(raw)

	proxyListen := env("CLIENT_GRPC_LISTEN", ":50052")
	go func() {
		if err := runGRPCProxy(proxyListen, grpcAddr); err != nil {
			log.Fatalf("grpc proxy error: %v", err)
		}
	}()

	// Conexión gRPC
	conn, err := grpc.Dial(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	check(err)
	defer conn.Close()
	client := pb.NewWeatherTweetServiceClient(conn)

	http.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Estamos vivos, desde el client-go"))
	})

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
```

## consumer kafka
El Kafka Consumer se encarga de recibir los mensajes de Kafka y procesarlos para almacenarlos en Valkey.

### Flujo de la comunicación:

#### Conexión a Kafka:

El consumer se conecta a Kafka utilizando los parámetros de configuración definidos en las variables de entorno. Esto incluye el broker, topic, y el consumer group.

#### Lectura de mensajes:

El consumer usa kafka-go (una librería de Go para interactuar con Kafka) para leer mensajes de un topic específico.

El código del consumer es responsable de "suscribirse" a ese topic de Kafka y empezar a leer mensajes en un bucle infinito.

#### Deserialización de mensajes:

Los mensajes que Kafka envía son generalmente en formato binario (aunque pueden estar en JSON o cualquier otro formato). El consumer deserializa esos mensajes en un objeto de tipo Clima.

Luego de la deserialización, el consumer verifica si los datos son correctos y completos.

#### Escritura en Valkey:

Después de que los datos son deserializados correctamente, se escriben en Valkey, la base de datos en memoria que estás utilizando.

La escritura en Valkey se hace tanto en un Hash como en un Stream. Los Hash contienen el estado actual del municipio (por ejemplo, temperatura, humedad, etc.), mientras que los Stream mantienen el historial de eventos.

Reintentos:

Si el consumer no puede deserializar el mensaje o escribir en Valkey, reintentará la operación.

#### Código del Kafka Consumer:

```go
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
	// Kafka configuration
	brokers := env("KAFKA_BROKER", "localhost:9092")
	topic := env("KAFKA_TOPIC", "clima")
	group := env("KAFKA_GROUP", "clima-consumer-group")

	log.Printf("Kafka consumer: brokers=%s topic=%s group=%s", brokers, topic, group)

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{brokers},
		Topic:   topic,
		GroupID: group,
	})

	// Redis (Valkey) client
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
```
## consumer rabbit
El RabbitMQ Consumer sigue un flujo similar al de Kafka, pero en lugar de leer desde un broker de Kafka, se conecta a RabbitMQ y lee los mensajes desde una cola de RabbitMQ.

### Flujo de la comunicación:

#### Conexión a RabbitMQ:

El consumer se conecta a RabbitMQ usando los parámetros de configuración (especificados en las variables de entorno).

### Lectura de mensajes:

El consumer utiliza la librería amqp091-go para conectarse a RabbitMQ y consumir los mensajes de la cola clima.

### Deserialización de los mensajes:

Los mensajes son deserializados desde su formato JSON para convertirlos en un objeto Clima.

### Escritura en Valkey:

Después de procesar el mensaje, el RabbitMQ Consumer escribe los datos deserializados en Valkey (similar a como lo hace el Kafka Consumer).

### Código del RabbitMQ Consumer:

```go
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
	// RabbitMQ connection
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

	// Redis (Valkey) client
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
		Values: map
```

## valkey
Valkey es una base de datos en memoria basada en Redis que se está utilizando para almacenar los datos sobre el clima que provienen de los consumidores de Kafka y RabbitMQ. Su función principal es almacenar el estado actual de los municipios y mantener un historial de los registros climáticos.

### Estructura de Valkey

Valkey está compuesto por dos tipos de estructuras en Redis:

#### Hashes: Se utilizan para almacenar el estado actual de los municipios.

Cada municipio tiene su propio hash con claves como name, temperature, humidity, weather y last_update.

Los valores de estas claves son dinámicos y se actualizan conforme se recibe nueva información de los consumidores de Kafka o RabbitMQ.

#### Streams: Se utilizan para almacenar el historial de registros.

Cada municipio tiene su propio stream en Redis, que guarda los registros de clima con su respectiva marca de tiempo (ts).

Estos streams permiten que se consulte el historial de datos de forma eficiente.

Flujo de datos hacia Valkey

### Lectura de datos desde Kafka o RabbitMQ:

Los consumers (de Kafka y RabbitMQ) leen los mensajes del topic o la queue respectivamente, que contienen los datos climáticos de un municipio.

Los mensajes se deserializan y se estructuran en un objeto Clima.

### Escritura en Redis (Valkey):

Una vez deserializado el mensaje, el consumer escribe la información en Valkey:

Estado actual: Se actualizan los hashes de los municipios, con los valores de temperatura, humedad, clima, y la última actualización.

### Historial: Se agrega una nueva entrada en el stream de cada municipio con los datos recibidos.

### Estructura de los datos almacenados en Valkey:

Hash:
```
name: Nombre del municipio.

temperature: Temperatura registrada.

humidity: Humedad registrada.

weather: Estado del clima (Soleado, Nublado, Lluvioso, etc.).

last_update: Fecha y hora de la última actualización de los datos.
```

Stream:
```
ts: Marca de tiempo de la lectura.

temperature: Temperatura registrada en ese momento.

humidity: Humedad registrada en ese momento.

weather: Clima registrado en ese momento.
```

### Interacciones con Valkey:

### Escritura de Datos:
Los datos de Kafka o RabbitMQ se envían a Valkey en la siguiente forma:

```go
func writeToValkey(rdb *redis.Client, c Clima) error {
    slug := slugify(c.Municipio) // Normaliza el nombre del municipio
    now := time.Now()

    // Escribir el estado actual en un HASH
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

    // (Opcional) TTL para los valores de estado
    _ = rdb.Expire(ctx, key, time.Hour)

    // Escribir el historial en un STREAM
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
```

### Consulta de Datos:
Los datos pueden ser consultados desde Valkey utilizando Redis, y se pueden obtener tanto el estado actual de un municipio como su historial.

Para obtener el estado actual de un municipio:

```go
weatherData, err := rdb.HGetAll(ctx, "municipality:mixco").Result()
```

Para obtener el historial de un municipio:

```go
historicalData, err := rdb.XRange(ctx, "clima:mixco", "-", "+").Result()

```

### Arquitectura y Configuración en el Cluster:

En tu cluster GKE, Valkey se está ejecutando como un contenedor dentro del namespace clima-app.

### Despliegue de Valkey:

El contenedor de Valkey se ha configurado para ser accesible en localhost:6379, y puede ser usado por los consumers de Kafka y RabbitMQ para almacenar datos.

### Configuración de Redis:

Valkey se conecta con la base de datos en memoria Redis que está corriendo dentro del contenedor. Esta configuración se ha establecido con la variable de entorno VALKEY_SERVICE_URL en los consumidores.

### Red de Kubernetes:

Los consumers (de Kafka y RabbitMQ) utilizan el servicio de Valkey para escribir los datos que leen de los brokers (Kafka y RabbitMQ) a la base de datos en memoria. Esto se logra mediante la configuración de red y de acceso dentro del cluster de Kubernetes.
## grafana
Grafana es una herramienta de visualización de datos ampliamente utilizada para crear dashboards y gráficos en tiempo real a partir de diversas fuentes de datos. En este proyecto, Grafana se utiliza para visualizar los datos almacenados en Valkey (Redis) y hacer análisis sobre el clima a través de los datos procesados por el sistema distribuido.

1. Descripción General de Grafana

Objetivo: Grafana conecta a Valkey (un servicio basado en Redis) para visualizar los datos meteorológicos almacenados por los consumidores de Kafka y RabbitMQ.

Dashboard: El dashboard que se crea en Grafana debe mostrar:

Últimos registros de clima por municipio.

Total de reportes por condición climática (barras).

Gráficas de temperatura y humedad promedio a lo largo del tiempo (líneas).

Grafana también permite configurar alertas basadas en ciertos umbrales de los datos, pero en este caso el objetivo es únicamente la visualización.

2. Instalación de Grafana en el Cluster

Grafana se puede instalar en el cluster de Kubernetes de GCP utilizando un Deployment y un Service. Esto facilita el acceso a la interfaz web de Grafana.

Pasos para instalar Grafana:

### Desplegar Grafana:

Para desplegar Grafana en el cluster de GKE, lo puedes hacer manualmente o utilizando Helm. Como ya tienes el servicio en Kubernetes, si no has instalado Grafana, puedes crear un archivo de deployment para Grafana en YAML. Aquí tienes un ejemplo básico:

```bash
apiVersion: apps/v1
kind: Deployment
metadata:
  name: grafana
  namespace: clima-app
  labels:
    app: grafana
spec:
  replicas: 1
  selector:
    matchLabels:
      app: grafana
  template:
    metadata:
      labels:
        app: grafana
    spec:
      containers:
      - name: grafana
        image: grafana/grafana:8.3.3
        ports:
        - containerPort: 3000
        env:
        - name: GF_SECURITY_ADMIN_PASSWORD
          value: "admin"  # Cambia esta contraseña en producción
        volumeMounts:
        - name: grafana-data
          mountPath: /var/lib/grafana
      volumes:
      - name: grafana-data
        emptyDir: {}
---
apiVersion: v1
kind: Service
metadata:
  name: grafana-service
  namespace: clima-app
  labels:
    app: grafana
spec:
  ports:
    - port: 80
      targetPort: 3000
  selector:
    app: grafana
  type: LoadBalancer
```

### Acceder a Grafana:

Una vez desplegado, Grafana estará accesible a través de un LoadBalancer. Puedes obtener la IP externa de Grafana con el siguiente comando:

kubectl get svc -n clima-app


Una vez que se obtenga la IP externa, accedes a Grafana en el navegador en http://<EXTERNAL-IP>:80.

### Login en Grafana:

El login de Grafana por defecto es:

`Username`: admin

`Password`: admin (o el valor configurado en GF_SECURITY_ADMIN_PASSWORD).

3. Configuración de la Fuente de Datos en Grafana

Agregar Redis como fuente de datos:
Para poder visualizar los datos en Grafana, necesitas agregar Valkey como fuente de datos, ya que Valkey se basa en Redis.

En el panel de Grafana, ve a Configuration (el ícono del engranaje) y selecciona Data Sources.

Haz clic en Add data source.

Elige Redis de la lista de fuentes disponibles.

En el campo URL, pon el servicio de Valkey que estás usando en el cluster, en este caso:

```
redis://valkey:6379
```

Configura cualquier parámetro adicional si es necesario.

### Configuración de Dashboards:

Una vez que tengas configurado Valkey como fuente de datos, podrás crear gráficos y visualizaciones.

Pasos para configurar un dashboard:

#### Últimos 10 registros por municipio:

En el panel de Grafana, selecciona + -> Dashboard.

Agrega un panel con el tipo de visualización Table.

Escribe una consulta en el Query para obtener los últimos 10 registros del municipio. Por ejemplo:

```
XREVRANGE clima:guatemala - + COUNT 10
```

Esto recuperará los últimos 10 registros del stream de Guatemala en Valkey.

`Gráfico de barras:` Total de reportes por condición climática:

Crea un panel con el tipo de visualización Bar Gauge.

Realiza una consulta que agrupe los datos por clima y cuente las entradas.

`Gráficas de líneas:` Temperatura y humedad promedio:

Crea un panel con visualización Time Series.

Realiza una consulta que calcule la temperatura promedio y la humedad promedio en función del ts (timestamp) de los registros de cada municipio.

4. Conexión entre Grafana y Valkey

Grafana usará Redis (Valkey) para obtener los datos de los registros y el estado actual de los municipios. Para ello, se realizan consultas sobre los streams y hashes en Redis, que se utilizan para poblar los diferentes gráficos en el dashboard.

`Stream:` Almacena el historial de los registros de clima por municipio.

`Hash:` Almacena el estado actual de cada municipio (última temperatura, humedad y estado del clima).

Ejemplo de consulta de Grafana a Redis:

XREVRANGE clima:guatemala - + COUNT 10


Esto devolverá los últimos 10 registros del municipio Guatemala, lo cual será utilizado para mostrarlo en el panel de tabla en Grafana.

5. Visualización de los Resultados en Grafana

En tu dashboard de Grafana, deberías poder ver gráficos que muestran:

La temperatura promedio de cada municipio a lo largo del tiempo.

La humedad promedio de cada municipio a lo largo del tiempo.

El total de reportes de clima (soleado, nublado, lluvioso, etc.) por municipio.


## Despliegue
Sobre el despliegue primero que nada tenemos la vm con zot, que es a en donde se cargaron todas las imagenes que son usadas en el cluster:
![alt text](image.png)

En este apartado pueden visualizarse las imagenes:
![alt text](image-1.png)

Como se menciono anteriormente, estas posteriormente son consumidas desde los archivos `.yaml`, los cuales las descargan mediante un link de ngrok que da a la maquina virtual con zot.

Al ejecutar todos los yamls:

```
└── 📁k8s
    ├── api-rust-ingress.yaml
    ├── api-rust.yaml
    ├── client-go.yaml
    ├── clima-app.yaml
    ├── grafana.yaml
    ├── kafka-consumer.yaml
    ├── kafka.yaml
    ├── namespace.yaml
    ├── rabbit-consumer.yaml
    ├── rabbitmq.yaml
    ├── server-go.yaml
    └── valkey-deployment.yaml
```

Estos comenzaran a correr en el cluster, por lo tanto, bastara con abrir locust y generar trafico para verificar la salida en grafana.

## extras
Para ver los logs de algun pod
```bash
#solo cambiarle el nombre
kubectl logs -n clima-app pod/api-rust-595f7f9449-66sws
```

Para ver todo

```bash
kubectl get all -n clima-app
```