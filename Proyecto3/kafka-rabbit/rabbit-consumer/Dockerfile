# syntax=docker/dockerfile:1

############################
# Build stage
############################
FROM golang:1.25.1-alpine AS builder
RUN apk add --no-cache git
WORKDIR /src

# 1) Dependencias (usa go.mod/go.sum de la RAÍZ del repo)
COPY go.mod go.sum ./
RUN go mod download

# 2) Código del consumer
COPY rabbit-consumer/ ./rabbit-consumer/

# 3) Compilar binario estático y pequeño
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-s -w" \
  -o /out/rabbit-consumer ./rabbit-consumer

############################
# Runtime stage
############################
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app

# Copiar binario
COPY --from=builder /out/rabbit-consumer /app/rabbit-consumer

# Este consumer no expone puertos; se conecta a Rabbit y Valkey.
# Defaults (puedes sobreescribir con -e en run / Deployment)
ENV RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/ \
    RABBIT_QUEUE=clima \
    VALKEY_SERVICE_URL=valkey:6379

CMD ["/app/rabbit-consumer"]
