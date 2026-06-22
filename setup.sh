#!/bin/bash

echo "=========================================================="
echo " A iniciar o Gestor de Logs - CM Oliveira do Hospital"
echo "=========================================================="

# Criar ficheiro .env central, caso não exista
if [ ! -f .env ]; then
    echo "A criar ficheiro .env central com credenciais padrão..."
    cat <<EOF > .env
# Configurações de Base de Dados
DB_USER=logs_admin
DB_PASS=CmaraOH_Logs_Secure_2026!
DB_NAME=syslogsdb
DB_HOST=db

# Configurações do Redis
REDIS_URL=redis:6379

#API Telegram

TG_BOT_TOKEN=coloque_aqui_o_token_do_seu_bot
EOF
    echo "Ficheiro .env criado com sucesso."
else
    echo "Ficheiro .env já existe. A prosseguir..."
fi

# Garantir permissões de execução para scripts internos, se existissem
chmod +x setup.sh

echo "A compilar e a levantar a infraestrutura em containers Docker..."
docker-compose up -d --build

echo "=========================================================="
echo " Sistema iniciado com sucesso!"
echo " - Painel de Administração disponível em: http://localhost"
echo " - Syslog Collector ativo nas portas: 514 (UDP e TCP)"
echo "=========================================================="