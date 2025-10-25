# syntax=docker/dockerfile:1

FROM golang:1.25.1-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app

# 1) Copiamos mod files desde la RAÍZ del repo (el build context debe ser server-go/)
COPY go.mod go.sum ./
RUN go mod download

# 2) Código necesario para compilar el server
COPY proto ./proto
COPY server ./server

# 3) Build binario estático
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-s -w" \
    -o /out/server-go ./server

# -------- runtime --------
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app

# Copiamos el binario
COPY --from=builder /out/server-go /app/server-go

# server-go EXPONE gRPC en 50051 para que client-go lo llame
EXPOSE 50051

# Defaults (cámbialos en K8s/Compose si hace falta)
# - WRITER_TARGET: both|kafka|rabbit|random
# - KAFKA_BROKER: host:port del broker
# - KAFKA_TOPIC:  nombre del topic
# - RABBITMQ_URL: amqp://user:pass@host:5672/
ENV WRITER_TARGET=both \
    KAFKA_BROKER=kafka:9092 \
    KAFKA_TOPIC=clima \
    RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/

CMD ["/app/server-go"]
