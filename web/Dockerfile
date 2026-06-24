FROM golang:1.24-alpine AS builder

# Instalar dependências do sistema
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Garantir que o Go transfere as dependências sem problemas de proxy
ENV GOPROXY=https://proxy.golang.org,direct

# Inicializar o módulo e descarregar as dependências
RUN go mod init syslog-web && \
    go get github.com/redis/go-redis/v9 && \
    go get github.com/lib/pq

# Copiar os ficheiros da raiz
COPY main.go .
COPY index.html .
# Copiar a nova diretoria de templates HTML
COPY templates/ ./templates/
COPY assets/ ./assets/
COPY app/ ./app/
COPY database/ ./database/
COPY models/ ./models/
COPY api/ ./api/
COPY mailer/ ./mailer/
# Compilar o binário de forma estática
RUN CGO_ENABLED=0 GOOS=linux go build -o web main.go

# --- Fase Final (Imagem Leve para Produção) ---
FROM alpine:latest

# Adicionar certificados e timezones
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copiar o binário compilado da fase anterior
COPY --from=builder /app/web .

COPY --from=builder /app/index.html .

# Copiar a pasta de templates na íntegra
COPY --from=builder /app/templates/ ./templates/
COPY --from=builder /app/assets/ ./assets/
COPY --from=builder /app/app/ ./app/
COPY --from=builder /app/database/ ./database/
COPY --from=builder /app/models/ ./models/
COPY --from=builder /app/api/ ./api/
COPY --from=builder /app/mailer/ ./mailer/
# Executar o serviço
CMD ["./web"]