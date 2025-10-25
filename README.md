# 202203009_LAB_SO1_2S2025

¡Hecho! Dejamos K8s para después. Aquí tienes un checklist cortito para probar todo en local (Rust → client-go → server-go → Kafka/Rabbit) usando tus binarios actuales.

1) Levanta Kafka y Rabbit en tu máquina

En la carpeta donde tienes el docker-compose.yaml:

```
docker compose up -d
docker ps   # verifica 9092 (Kafka) y 5672/15672 (Rabbit)
```

2) Arranca server-go (publica a ambos brokers)

Terminal A:
```
cd server-go/server
export WRITER_TARGET=both
export KAFKA_BROKER=localhost:9092
export RABBITMQ_URL=amqp://guest:guest@localhost:5672/
go run main.go
```

Deberías ver: Go gRPC server escuchando en :50051 ...

3) Arranca client-go (proxy gRPC y endpoint HTTP opcional)

Terminal B:

```
cd ../client
export WRITER_ADDR=localhost:50051
export CLIENT_GRPC_LISTEN=:50052
export PORT=8081
go run main.go
```

Verás algo tipo: client-go gRPC proxy escuchando en :50052 → upstream=localhost:50051

4) Arranca la API de Rust

Terminal C:

```
cd ../../api-rust
export GRPC_SERVER_URL=http://localhost:50052
cargo run
```

Verás: Conectando a gRPC en: http://localhost:50052 y el server web en :8080.

5) Arranca tus consumers (opcional pero recomendable)

Terminal D (Kafka):

```
cd ../kafka-rabbit/kafka-consumer
go run consumer-kafka.go
```

Terminal E (Rabbit):

```
cd ../rabbit-consumer
go run consumer-rabbit.go
```

6) Prueba end-to-end

Envía un POST a la API de Rust (usa enteros porque tu Rust mapea enums por número):

```
curl -X POST http://localhost:8080/clima \
  -H 'Content-Type: application/json' \
  -d '{"municipality":1,"temperature":22,"humidity":60,"weather":1}'
```

municipality: 1=mixco, 2=guatemala, 3=amatitlan, 4=chinautla

weather: 1=sunny, 2=cloudy, 3=rainy, 4=foggy

¿Qué deberías ver?

Rust: “Recibió JSON” y “Respuesta gRPC…”

client-go: (el proxy no imprime por cada llamada, es normal)

server-go: 📦 Recibido por gRPC: ... → JSON=...

Consumers:

Kafka: ✅ [Kafka] Mensaje: {...}

Rabbit: ✅ [RabbitMQ] Mensaje recibido: {...}

Tips rápidos si algo falla

Error “lookup rabbitmq: no such host”: asegurarte de haber puesto RABBITMQ_URL=amqp://guest:guest@localhost:5672/ en server-go (si ves rabbitmq:5672 es que estás usando el valor de K8s por defecto).

Error gRPC desde Rust: confirma que client-go está en :50052 y que GRPC_SERVER_URL apunta a http://localhost:50052.

Kafka sin mensajes: confirma que server-go sigue corriendo y que KAFKA_BROKER=localhost:9092. Si el tópico no existe y tu broker no lo autocrea, créalo (con utilidades de Kafka) o reinicia con autocreación habilitada.

Puerto ocupado: cambia CLIENT_GRPC_LISTEN (p. ej. :50053) y ajusta GRPC_SERVER_URL a http://localhost:50053.

Cuando termines esta prueba local y te pasen el archivo de guía, lo adaptamos a K8s/GCP en un toque.

---


Si server-go corre en tu host (con go run):
export KAFKA_BROKER=localhost:9092
export RABBITMQ_URL=amqp://guest:guest@localhost:5672/
export KAFKA_TOPIC=clima
export WRITER_TARGET=both      # o 'rabbit' si quieres ignorar Kafka mientras pruebas
go run server/main.go

Si server-go corre en un contenedor del mismo compose:
export KAFKA_BROKER=kafka:29092
export RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/
export KAFKA_TOPIC=clima
export WRITER_TARGET=both

---
## Creacion de cluster
Para la parte de la creacion del cluster basto con ejecutar esto en la consola:

```bash
# 1) Selecciona el proyecto y la zona
gcloud config set project nombreProyecto-id-proyecto
gcloud config set compute/zone us-west1-a

# 2) Asegura la API de GKE
gcloud services enable container.googleapis.com

# 3) Crea el clúster (Standard, zonal)
gcloud container clusters create proyecto \
  --zone us-west1-a \
  --num-nodes 4 \
  --machine-type e2-medium \
  --disk-type pd-standard \
  --disk-size 25 \
  --tags allin,allout \
  --image-type COS_CONTAINERD \
  --release-channel regular \
  --enable-autoupgrade \
  --enable-autorepair \
  --no-enable-network-policy \
  --enable-ip-alias

```

Vamos a conectar y verificar con esto:
```
gcloud container clusters get-credentials proyecto --zone us-west1-a --project proyecto3-476206
kubectl get nodes
```

Ahora creamos el namespace:


---
Se subieron las imagenes de la api de rust, del server, client y los consumers en docker hacia zot.

