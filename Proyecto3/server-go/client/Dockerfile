# Multi-stage build para optimizar el tamaño de la imagen
FROM golang:1.25.1-alpine AS builder

# Instalar git (necesario para algunas dependencias de Go)
RUN apk add --no-cache git

# Crear directorio de trabajo
WORKDIR /app

# Copiar archivos de módulo Go
COPY go.mod go.sum ./

# Descargar dependencias
RUN go mod download

# Copiar código fuente
COPY . .

# 3) Build
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-s -w" \
    -o /out/client-go ./client
# Imagen final más pequeña
FROM alpine:latest

# Instalar certificados CA para conexiones HTTPS/TLS
RUN apk --no-cache add ca-certificates

# Crear directorio de trabajo
WORKDIR /root/

# Importante: copiar desde /out (no /app)
COPY --from=builder /out/client-go /app/client-go

# Documentamos los puertos:
#  - 8081: HTTP (health y /tweet opcional)
#  - 50052: gRPC server del proxy
EXPOSE 8081 50052

# Defaults seguros (puedes cambiarlos en el Deployment)
ENV PORT=8081 \
    CLIENT_GRPC_LISTEN=:50052

# Comando para ejecutar la aplicación
CMD ["/app/client-go"]