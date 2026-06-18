FROM golang:1.24-alpine AS builder

# 1. Instalar git e certificados (previne erros de SSL em redes corporativas)
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# 2. Forçar o Go a usar ligação direta caso o proxy oficial esteja bloqueado
ENV GOPROXY=https://proxy.golang.org,direct

# Inicializar módulo Go e instalar dependência do Redis
RUN go mod init syslog-web && \
    go get github.com/redis/go-redis/v9 && \
    go get github.com/lib/pq

COPY main.go .
COPY index.html .
COPY script.js .
# Compilar binário estático
RUN CGO_ENABLED=0 GOOS=linux go build -o web main.go

FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/web .
COPY --from=builder /app/index.html .
COPY --from=builder /app/script.js .
CMD ["./web"]