# SyslogCollector

Centralized Syslog Collection, Processing and Monitoring Platform built with Go, Redis, PostgreSQL and HTMX.

## Overview

SyslogCollector is a lightweight observability platform designed to collect, normalize, process and visualize Syslog events from heterogeneous environments.

The platform supports both modern and legacy Syslog formats, making it suitable for:

* Windows Servers
* Linux Servers
* Firewalls
* Switches
* Routers
* Network Appliances
* Security Devices
* Embedded Systems

Instead of storing raw logs directly in a database, SyslogCollector uses a queue-based architecture that decouples ingestion from persistence, increasing reliability and resilience under heavy workloads.

---

## Architecture

```text
                  ┌───────────────┐
                  │ Syslog Clients│
                  └───────┬───────┘
                          │
               UDP/TCP 514│
                          ▼
               ┌─────────────────┐
               │    Collector    │
               │ RFC3164 RFC5424 │
               └────────┬────────┘
                        │
                        ▼
               ┌─────────────────┐
               │      Redis      │
               │ Event Queue     │
               └────────┬────────┘
                        │
                        ▼
               ┌─────────────────┐
               │   Web Worker    │
               │ Persistence     │
               └────────┬────────┘
                        │
                        ▼
               ┌─────────────────┐
               │   PostgreSQL    │
               └────────┬────────┘
                        │
                        ▼
               ┌─────────────────┐
               │ HTMX Dashboard  │
               └─────────────────┘
```

---

## Technology Stack

### Backend

* Go
* PostgreSQL
* Redis

### Frontend

* HTMX
* Tailwind CSS v4
* Lucide Icons
* Chart.js

### Infrastructure

* Docker
* Docker Compose
* Nginx

The platform is composed of independent services running in containers. Redis acts as an event buffer and PostgreSQL stores normalized log records.

---

## Current Features

### Log Collection

* Syslog UDP (514)
* Syslog TCP (514)
* RFC3164 support
* RFC5424 support
* Automatic normalization

### Processing

* Queue-based ingestion using Redis
* Asynchronous persistence
* Automatic field extraction
* Severity normalization
* Facility normalization

### Dashboard

* Real-time log monitoring
* Search and filtering
* Severity filtering
* Detailed log inspection drawer
* CSV export

### Analytics

* Severity distribution charts
* Top active hosts
* Operational statistics

### Administration

* Authentication system
* Retention policy management
* Administrative settings
* System tools panel

---

## Project Structure

```text
SyslogCollector
├── collector/
│   ├── main.go
│   ├── Dockerfile
│   └── go.mod
│
├── web/
│   ├── main.go
│   ├── index.html
│   ├── script.js
│   ├── style.css
│   ├── output.css
│   └── Dockerfile
│
├── nginx/
│   └── default.conf
│
├── compose.yml
├── .env.example
└── README.md
```

---

## Installation

### Clone Repository

```bash
git clone https://github.com/ghvalentim/SyslogCollector.git
cd SyslogCollector
```

### Create Environment File

```bash
cp .env.example .env
```

Configure:

```env
DB_USER=postgres
DB_PASS=password
DB_NAME=syslog
DB_HOST=db

REDIS_URL=redis:6379
```

Based on the provided environment template.

### Start Services

```bash
docker compose up -d --build
```

---

## Development Roadmap

### Sprint 1 — Core Collector

* UDP listener
* TCP listener
* Redis integration
* RFC3164 support
* RFC5424 support

### Sprint 2 — Persistence Layer

* PostgreSQL integration
* Log normalization
* Queue workers
* Retention policies

### Sprint 3 — Dashboard

* HTMX frontend
* Live updates
* Filtering
* Search

### Sprint 4 — Analytics

* Severity statistics
* Host statistics
* Dashboard charts

### Sprint 5 — Administrative Panel

* Authentication
* Settings management
* System tools
* CSV export

### Sprint 6 — Modern UI

* Tailwind CSS v4
* Responsive dashboard
* Improved navigation
* Enhanced log details drawer

### Sprint 7 — Log Collection Policies (Planned)

Goal:

Introduce configurable collection policies per deployment.

Features:

* Minimum severity level
* Ignored applications
* Ignored hosts
* Ignored keywords
* Dynamic policy updates
* Collection statistics
* Log filtering before queue insertion

Future flow:

```text
Receive Log
      ↓
Parser
      ↓
Policy Engine
      ↓
Redis
      ↓
Database
```

---

## Long-Term Vision

The project is evolving beyond a simple Syslog server.

Current direction:

```text
Syslog Server
      ↓
Log Collector
      ↓
Observability Platform
      ↓
Lightweight SIEM
```

Planned future capabilities:

* Policy engine
* Alerting system
* Correlation rules
* Threat indicators
* Event prioritization
* Multi-tenant deployments
* Security dashboards
* Notification integrations

---

## License

This project is currently distributed as open source.

Feel free to fork, improve and contribute.

---

## Author

Gabriel Valentim

Computer Systems and Programming Student (TPSI)

Portugal
