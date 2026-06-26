FROM golang:1.26-alpine AS builder

# Instalar dependências do sistema
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Garantir que o Go transfere as dependências sem problemas de proxy
ENV GOPROXY=https://proxy.golang.org,direct

# Inicializar o módulo e descarregar as dependências
RUN go mod init syslog-web && \
    go get github.com/redis/go-redis/v9 && \
    go get github.com/lib/pq && \
    go get github.com/go-telegram-bot-api/telegram-bot-api/v5


# Copiar os ficheiros da raiz
COPY main.go .
# Copiar a nova diretoria de templates HTML
COPY public/ ./public/
COPY app/ ./app/
COPY data/ ./data/
COPY models/ ./models/
COPY api/ ./api/
# Compilar o binário de forma estática
RUN CGO_ENABLED=0 GOOS=linux go build -o web main.go

# --- Fase Final (Imagem Leve para Produção) ---
FROM alpine:latest

# Adicionar certificados e timezones
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copiar o binário compilado da fase anterior
COPY --from=builder /app/web .

# Copiar a pasta de templates na íntegra
COPY --from=builder /app/public/ ./public/
COPY --from=builder /app/app/ ./app/
COPY --from=builder /app/data/ ./data/
COPY --from=builder /app/models/ ./models/
COPY --from=builder /app/api/ ./api/
# Executar o serviço
CMD ["./web"]