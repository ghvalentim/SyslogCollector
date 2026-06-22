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
![Worker Pool](https://img.shields.io/badge/Worker%20Pool-Enabled-success)
![Classification](https://img.shields.io/badge/Event%20Classification-Enabled-success)
![Alerts](https://img.shields.io/badge/Alert%20Management-Enabled-orange)

![Status](https://img.shields.io/badge/Status-Active%20Development-brightgreen)
![Version](https://img.shields.io/badge/Version-v0.8.0--alpha-blue)
![License](https://img.shields.io/badge/License-MIT-yellow)

---

## 📌 Visão Geral

O SyslogCollector é uma plataforma de observabilidade leve desenvolvida para recolha, processamento, classificação e visualização de eventos Syslog em pequenas e médias infraestruturas.

### Funcionalidades

- Receção Syslog UDP e TCP
- Suporte RFC3164 e RFC5424
- Dashboard Web em tempo real
- Classificação automática de eventos
- Facilities humanizadas
- Motor de políticas dinâmico
- Redis Pub/Sub
- Gestão de alertas
- Estatísticas operacionais
- Exportação CSV
- Autenticação integrada
- Docker Compose Ready

---

## 🏗 Arquitetura

```text
Clientes Syslog
(Firewalls, Linux, Windows, Switches, APs)

            │
            ▼

     Syslog Collector
      UDP/TCP :514

            │
            ▼

      RFC3164 Parser
      RFC5424 Parser

            │
            ▼

    Event Classifier
    Facility Mapper

            │
            ▼

      Policy Engine

            │
            ▼

       Redis Queue

            │
            ▼

      Worker Pool

            │
            ▼

       PostgreSQL

            │
            ▼

      Dashboard Web
````

## 🚀 Tecnologias

### Backend

* Go
* PostgreSQL
* Redis

### Frontend

* HTMX
* Tailwind CSS
* Chart.js
* Lucide Icons

### Infraestrutura

* Docker
* Docker Compose
* Nginx

---

## 📥 Recolha de Logs

Suporta:

* Syslog UDP
* Syslog TCP
* RFC3164
* RFC5424

Campos processados:

* Timestamp
* Source IP
* Hostname
* Application
* Facility
* Facility Name
* Severity
* Payload
* Source Type

---

## 🧠 Classificação Automática

O sistema classifica automaticamente os eventos recebidos.

Categorias atualmente suportadas:

* Firewall
* Windows
* Linux
* DNS
* DHCP
* Network
* Web
* Security
* Database
* Unknown

A classificação considera:

* Application Name
* Hostname
* Facility
* Conteúdo da mensagem

---

## 🏷 Facilities Humanizadas

Conversão automática das facilities Syslog.

| Código | Nome   |
| ------ | ------ |
| 0      | Kernel |
| 1      | User   |
| 2      | Mail   |
| 3      | Daemon |
| 4      | Auth   |
| 16     | Local0 |
| 17     | Local1 |
| 18     | Local2 |
| 19     | Local3 |
| 20     | Local4 |
| 21     | Local5 |
| 22     | Local6 |
| 23     | Local7 |

---

## 🛡 Motor de Políticas

Permite filtragem dinâmica sem reiniciar o collector.

### Severidade mínima

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

### Hosts ignorados

```text
printer01
switch-lab
test-server
```

### Aplicações ignoradas

```text
nginx
systemd
dnsmasq
```

### Palavras-chave ignoradas

```text
heartbeat
healthcheck
favicon.ico
```

---

## 🔄 Atualização Dinâmica

As alterações são sincronizadas via Redis Pub/Sub.

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
Reload Automático
```

Sem necessidade de reiniciar containers.

---

## 🚨 Gestão de Alertas

Já implementado:

* CRUD de alertas
* Persistência PostgreSQL
* Interface administrativa
* Configuração dinâmica

Preparado para:

* Email
* Telegram
* Discord
* Webhooks

Exemplos:

* Mais de 20 erros em 5 minutos
* Mais de 50 eventos Firewall em 10 minutos
* Eventos contendo palavras-chave específicas

---

## 📊 Dashboard

Funcionalidades:

* Atualização automática
* Pesquisa textual
* Filtro por severidade
* Drawer de detalhes
* Exportação CSV
* Navegação HTMX
* Interface responsiva

---

## 📈 Estatísticas

Disponíveis:

* Logs recebidos
* Logs armazenados
* Logs filtrados
* Hosts mais ativos
* Aplicações mais ativas
* Distribuição por severidade
* Distribuição por origem

---

## 🔐 Autenticação

Autenticação baseada em sessão.

Configuração via painel administrativo.

---

## 🗄 Retenção Automática

Suporte a retenção configurável.

Exemplo:

```text
30 dias
```

Os registos antigos são removidos automaticamente.

---

## 📦 Instalação

### Clonar

```bash
git clone https://github.com/ghvalentim/SyslogCollector.git

cd SyslogCollector
```

### Configurar

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

### Iniciar

```bash
docker compose up -d --build
```

---

## 🌐 Acesso

```text
http://localhost
```

---

## 📁 Estrutura do Projeto

```text
SyslogCollector
│
├── collector
│   ├── main.go
│   └── app
│       ├── classifier.go
│       ├── facility.go
│       ├── listener.go
│       ├── models.go
│       ├── parser.go
│       ├── policy.go
│       ├── redis.go
│       └── workers.go
│
├── web
│   ├── main.go
│   ├── app
│   │   ├── alerts.go
│   │   ├── auth.go
│   │   ├── database.go
│   │   ├── handlers.go
│   │   ├── models.go
│   │   ├── policies.go
│   │   ├── routes.go
│   │   ├── services.go
│   │   ├── settings.go
│   │   └── stats.go
│   │
│   ├── assets
│   │   ├── output.css
│   │   └── script.js
│   │
│   └── templates
│       ├── alerts.html
│       ├── login.html
│       ├── logs.html
│       ├── policies.html
│       ├── settings.html
│       ├── stats.html
│       └── tools.html
│
├── nginx
├── compose.yml
├── setup.sh
└── README.md
```

---

## 🗺 Roadmap

### Sprint 1 ✅

* Receção UDP/TCP
* PostgreSQL
* Redis Queue

### Sprint 2 ✅

* Dashboard HTMX
* Pesquisa
* Exportação CSV

### Sprint 3 ✅

* Autenticação
* Configurações

### Sprint 4 ✅

* Estatísticas
* Gráficos

### Sprint 5 ✅

* Drawer de detalhes
* Melhorias UI

### Sprint 6 ✅

* Tailwind local
* Refatoração visual

### Sprint 7 ✅

* Motor de Políticas
* Redis Pub/Sub
* Filtragem dinâmica

### Sprint 8 🚧

* ✅ Classificação automática
* ✅ Facilities humanizadas
* ✅ Worker Pool
* ✅ CRUD de alertas
* ✅ Estatísticas por origem
* 🚧 Alert Engine
* 🚧 Notificações

### Sprint 9

* Correlação de eventos
* Email
* Telegram
* Discord
* Webhooks

### Sprint 10

* Multiutilizador
* Auditoria
* Perfis de acesso

### Sprint 11

* Kubernetes
* Alta disponibilidade
* Clustering

---

## 📄 Licença

MIT License.

---

## 👨‍💻 Autor

Gabriel Valentim

Projeto de estudo focado em:

* Observabilidade
* Redes
* Syslog
* Golang
* HTMX
* Arquitetura de Sistemas

---

**SyslogCollector v0.8.0-alpha**
*Observabilidade simples, rápida e sem complicações.*
