FROM golang:1.26-alpine AS builder

# 1. Instalar git e certificados (previne erros de SSL em redes corporativas)
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# 2. Forçar o Go a usar ligação direta caso o proxy oficial esteja bloqueado
ENV GOPROXY=https://proxy.golang.org,direct

# Inicializar módulo Go e instalar dependência do Redis
RUN go mod init syslog-collector && \
    go get github.com/redis/go-redis/v9 && \
    go get github.com/influxdata/go-syslog/v3

COPY main.go .    
COPY models/ ./models/
COPY redis/ ./redis/
COPY settings/ ./settings/
COPY tools/ ./tools/
# Compilar binário estático
RUN CGO_ENABLED=0 GOOS=linux go build -o server main.go

FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/server .
COPY --from=builder /app/models/ ./models/
COPY --from=builder /app/redis/ ./redis/
COPY --from=builder /app/settings/ ./settings/
COPY --from=builder /app/tools/ ./tools/

CMD ["./server"]