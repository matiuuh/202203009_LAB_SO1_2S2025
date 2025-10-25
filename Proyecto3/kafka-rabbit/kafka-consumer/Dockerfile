# syntax=docker/dockerfile:1

############################
# Build stage
############################
FROM golang:1.25.1-alpine AS builder
RUN apk add --no-cache git
WORKDIR /src

# 1) Dependencias (usa el go.mod/go.sum de la RAÍZ del repo)
COPY go.mod go.sum ./
RUN go mod download

# 2) Código del consumer
COPY kafka-consumer/ ./kafka-consumer/

# 3) Compilar binario estático
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-s -w" \
  -o /out/kafka-consumer ./kafka-consumer

############################
# Runtime stage
############################
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app

# Copiar binario
COPY --from=builder /out/kafka-consumer /app/kafka-consumer

# Este consumer NO expone puertos (sólo se conecta a Kafka y Valkey)
# Variables por defecto (puedes sobreescribirlas con -e al correr)
ENV KAFKA_BROKER=kafka:9092 \
    KAFKA_TOPIC=clima \
    KAFKA_GROUP=clima-consumer-group \
    VALKEY_SERVICE_URL=valkey:6379

CMD ["/app/kafka-consumer"]
