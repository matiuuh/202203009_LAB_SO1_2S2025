FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY main.go .
RUN go build -o /out/valkey-smoke .

FROM alpine:3.19
COPY --from=builder /out/valkey-smoke /valkey-smoke
ENTRYPOINT ["/valkey-smoke"]
