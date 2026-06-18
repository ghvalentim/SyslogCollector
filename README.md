# 🛡️ SyslogCollector

![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-17-4169E1?logo=postgresql&logoColor=white)
![Redis](https://img.shields.io/badge/Redis-8-DC382D?logo=redis&logoColor=white)
![HTMX](https://img.shields.io/badge/HTMX-1.9+-3366CC?logo=htmx&logoColor=white)
![TailwindCSS](https://img.shields.io/badge/TailwindCSS-4.x-06B6D4?logo=tailwindcss&logoColor=white)
![Chart.js](https://img.shields.io/badge/Chart.js-4.x-FF6384?logo=chartdotjs&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker&logoColor=white)
![Nginx](https://img.shields.io/badge/Nginx-Reverse%20Proxy-009639?logo=nginx&logoColor=white)

![RFC3164](https://img.shields.io/badge/Syslog-RFC3164-success)
![RFC5424](https://img.shields.io/badge/Syslog-RFC5424-success)
![Redis Pub/Sub](https://img.shields.io/badge/Redis-Pub%2FSub-orange)
![Policy Engine](https://img.shields.io/badge/Policy%20Engine-Dynamic-blueviolet)

![Status](https://img.shields.io/badge/Status-Active%20Development-brightgreen)
![Version](https://img.shields.io/badge/Version-v0.7.0-blue)
![License](https://img.shields.io/badge/License-MIT-yellow)

Sistema moderno de recolha, processamento, filtragem e observabilidade de eventos Syslog.

Desenvolvido em Go, PostgreSQL, Redis, HTMX e Tailwind CSS.

---

## 📌 Visão Geral

O SyslogCollector é uma plataforma de observabilidade leve destinada a pequenas e médias infraestruturas.

O sistema permite:

- Receber logs Syslog via UDP e TCP
- Interpretar mensagens RFC3164 e RFC5424
- Aplicar políticas de filtragem em tempo real
- Armazenar eventos em PostgreSQL
- Visualizar eventos através de dashboard web responsivo
- Exportar registos para CSV
- Gerir retenção de dados
- Obter estatísticas operacionais
- Administrar o sistema através de interface web

---

## 🏗 Arquitetura

```text
Dispositivos
(Firewalls, Switches, Linux, Windows, APs)

            │
            ▼

      Syslog Collector
      UDP/TCP :514

            │
            ▼

     Motor de Políticas
       (em memória)

            │
            ▼

          Redis
      Queue + Pub/Sub

            │
            ▼

        Web Worker

            │
            ▼

       PostgreSQL

            │
            ▼

     Dashboard HTMX
```

---

## 🚀 Tecnologias

### Backend

- Go
- PostgreSQL
- Redis

### Frontend

- HTMX
- Tailwind CSS
- Chart.js
- Lucide Icons

### Infraestrutura

- Docker
- Docker Compose
- Nginx

---

# Funcionalidades

## 📥 Recolha de Logs

Suporte para:

- Syslog UDP
- Syslog TCP
- RFC3164
- RFC5424

Campos normalizados:

- Timestamp
- IP Origem
- Protocolo
- Hostname
- Aplicação
- Severidade
- Facility
- Payload

---

## 📊 Dashboard em Tempo Real

Visualização contínua dos eventos.

Recursos:

- Pesquisa textual
- Filtro por severidade
- Atualização automática
- Pausa de atualização
- Drawer de detalhes
- Exportação CSV

---

## 📈 Estatísticas

Painel analítico com:

- Logs recebidos
- Logs armazenados
- Logs filtrados
- Distribuição por severidade
- Hosts mais ativos

---

## 🛡️ Motor de Políticas Dinâmicas

As políticas são aplicadas antes da persistência.

### Filtros disponíveis

#### Severidade mínima

Exemplo:

```text
Erro
```

Ignora:

```text
Aviso
Info
Debug
```

---

#### Aplicações ignoradas

```text
nginx
dnsmasq
systemd
```

---

#### Hosts ignorados

```text
printer01
switch-lab
test-server
```

---

#### Palavras-chave ignoradas

```text
healthcheck
heartbeat
GET /favicon.ico
```

---

### Atualização em tempo real

Fluxo:

```text
Dashboard
    │
    ▼

PostgreSQL
    │
    ▼

Redis
SET + Publish
    │
    ▼

Collector
Reload automático
```

Não é necessário reiniciar containers.

---

## 🔐 Autenticação

Autenticação integrada.

Configurações armazenadas:

- Utilizador
- Palavra-passe

Sessões protegidas por cookie.

---

## 🗄 Retenção Automática

Configuração de retenção em dias.

Exemplo:

```text
30 dias
```

Processo automático executado periodicamente.

---

## 📦 Instalação

### Clonar

```bash
git clone https://github.com/ghvalentim/SyslogCollector.git
cd SyslogCollector
```

---

### Configurar ambiente

```bash
cp .env.example .env
```

Editar:

```env
DB_HOST=postgres
DB_NAME=syslog
DB_USER=syslog
DB_PASS=syslog

REDIS_URL=redis:6379
```

---

### Iniciar

```bash
docker compose up -d
```

---

## 🌐 Acesso

Dashboard:

```text
http://localhost:8080
```

---

## 📁 Estrutura

```text
SyslogCollector
│
├── collector/
│   └── main.go
│
├── web/
│   ├── main.go
│   ├── index.html
│   ├── script.js
│   ├── style.css
│   └── output.css
│
├── nginx/
│   └── default.conf
│
├── compose.yml
├── setup.sh
└── README.md
```

---

# Roadmap

## Sprint 1

- Recolha UDP
- Recolha TCP
- PostgreSQL
- Redis Queue

## Sprint 2

- Dashboard HTMX
- Pesquisa
- Exportação CSV

## Sprint 3

- Autenticação
- Definições

## Sprint 4

- Estatísticas
- Gráficos

## Sprint 5

- Drawer de detalhes
- Interface profissional

## Sprint 6

- Tailwind compilado localmente
- Otimizações de UI

## Sprint 7 ✅

- Motor de Políticas
- Redis Pub/Sub
- Filtros dinâmicos
- Estatísticas de filtragem

## Sprint 8 (Planeada)

- Classificação automática de origem
- Regras por Facility
- Tags automáticas
- Alertas

## Sprint 9 (Planeada)

- Multiutilizador
- Auditoria
- Perfis de acesso

## Sprint 10 (Planeada)

- Kubernetes
- Alta disponibilidade
- Clustering

---

# Licença

Projeto desenvolvido para fins académicos, laboratoriais e de aprendizagem em observabilidade, redes e desenvolvimento de software.

---

**SyslogCollector**
Observabilidade simples, rápida e sem complicações.