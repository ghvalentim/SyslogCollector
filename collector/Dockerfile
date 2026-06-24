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
COPY app ./app/
# Compilar binário estático
RUN CGO_ENABLED=0 GOOS=linux go build -o collector main.go

FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/collector .
COPY --from=builder /app/app/ ./app/

CMD ["./collector"]