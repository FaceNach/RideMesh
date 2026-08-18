# RideShare — Microservices Starter

A ride-sharing platform built as an event-driven microservices system. Riders request trips, drivers register and get matched through geohash-based packages, the trip service computes routes and fare estimates, and payments are handled end-to-end with Stripe — all in real time over WebSockets.

**Read this README in:** [English](#english) · [Español](#español)

---

## English

### What is this?

RideShare is a full-stack, event-driven ride-sharing application used as a starter template for building Go microservices. It demonstrates a complete workflow:

1. A rider previews a trip (route + fare packages) and starts it.
2. The trip service finds available drivers through RabbitMQ events.
3. Drivers accept or reject trip requests over WebSockets.
4. The rider pays through a Stripe Checkout session.
5. A successful payment finalizes the trip, all coordinated via message queues.

The backend is written in Go with gRPC for synchronous service-to-service calls and RabbitMQ for asynchronous event-driven communication. The frontend is a Next.js application with real-time maps.

### Architecture

```
                 +-----------------------+
                 |    Next.js Web App    |
                 |      (port 3000)      |
                 +-----------+-----------+
                             | HTTP / WebSocket
                 +-----------v-----------+
                 |     API Gateway       |
                 |  REST + WS (port 8081)|
                 +---+-------+-------^---+
                     |       |       |
              gRPC   |       |       |  events (RabbitMQ)
        +------------+       |       +------------+
        |                    |                    |
+-------v-------+   +--------v-------+   +--------v---------+
| Trip Service  |   | Driver Service |   |  Payment Service |
| gRPC :9093    |   | gRPC :9092     |   |      :9004       |
| MongoDB + OSRM|   | geohash match  |   |     Stripe       |
+-------+-------+   +----------------+   +------------------+
        |  events
+-------v-------+
|   RabbitMQ    |   <-- event bus with DLX
+---------------+

+--------+----------+----------+----------+
| Jaeger |  MongoDB |  Stripe  |  OSRM    |
| :16686 |          | (external)| (external)|
+--------+----------+----------+----------+
```

All Go services report traces to Jaeger through OpenTelemetry.

### Tech Stack

| Layer | Tools |
| --- | --- |
| Backend | Go 1.23, gRPC, Protocol Buffers, gorilla/websocket, google/uuid, mmcloughlin/geohash |
| Messaging | RabbitMQ (AMQP) with Dead Letter Exchange and retries |
| Data | MongoDB (mongo-driver) |
| Payments | Stripe (stripe-go v81, Checkout sessions, webhooks) |
| Observability | OpenTelemetry SDK + Jaeger exporter (gRPC, HTTP and RabbitMQ instrumentation) |
| Frontend | Next.js 15, React 19, TypeScript, Tailwind CSS, Leaflet / react-leaflet, Stripe.js, Radix UI |
| Infra | Docker, Kubernetes, Tilt (local dev), Makefile |
| Tooling | `tools/create_service.go` scaffolding script, `protoc` code generation |

### Project Structure

```
.
├── build/                          # Compiled service binaries (gitignored)
├── infra/
│   ├── development/
│   │   ├── docker/                 # Dockerfiles + build scripts per service
│   │   └── k8s/                    # Dev manifests: deployments, config, secrets, jaeger, rabbitmq
│   └── production/
│       ├── docker/                 # Production Dockerfiles
│       └── k8s/                    # Production manifests (api-gateway, trip-service)
├── proto/                          # .proto definitions (trip, driver)
├── services/
│   ├── api-gateway/                # HTTP/REST + WebSockets entry point (port 8081)
│   ├── driver-service/             # Driver registration + geohash matching (port 9092)
│   ├── payment-service/            # Stripe payments (port 9004)
│   └── trip-service/               # Routes, fares, trip lifecycle (port 9093)
├── shared/                         # Shared Go library
│   ├── contracts/                  # AMQP / HTTP / WS payload contracts
│   ├── db/                         # MongoDB client helpers
│   ├── env/                        # Environment variable helpers
│   ├── messaging/                  # RabbitMQ wrapper: exchanges, queues, DLX, retries
│   ├── proto/                      # Generated gRPC code (driver, trip)
│   ├── retry/                      # Retry utilities
│   ├── tracing/                    # OpenTelemetry setup + interceptors
│   ├── types/                      # Shared types
│   └── util/                       # Utilities
├── tools/
│   └── create_service.go           # Scaffolds a new clean-architecture service
├── web/                            # Next.js frontend (port 3000)
├── Makefile                        # proto code generation
└── Tiltfile                        # Local dev environment definition
```

The Go services follow **Clean Architecture**: `cmd/` (entrypoint), `internal/domain/` (business models and interfaces), `internal/service/` (business logic), `internal/infrastructure/` (repositories, events, gRPC, external APIs) and `pkg/types/` (shared public types).

### Services

| Service | Port | Responsibility |
| --- | --- | --- |
| `web` | 3000 | Next.js UI: rider/driver maps, package selector, Stripe checkout |
| `api-gateway` | 8081 | REST + WebSocket gateway; proxies gRPC to services; Stripe webhook receiver |
| `driver-service` | 9092 | gRPC: driver register/unregister; geohash-based driver matching; trip request consumer |
| `trip-service` | 9093 | gRPC: trip preview/create; route fetching (OSRM), fare estimation; trip lifecycle + event publisher |
| `payment-service` | 9004 | Stripe Checkout sessions and webhook handling via RabbitMQ consumers |

**API Gateway endpoints:**

- `POST /trip/preview` — preview route and fare packages
- `POST /trip/start` — start a trip with a selected fare
- `WS /ws/drivers` — real-time stream for drivers (trip requests, assignments)
- `WS /ws/riders` — real-time stream for riders (driver matching updates)
- `POST /webhook/stripe` — Stripe payment webhook

### RabbitMQ Events

Event-driven flow between services over the `trip` exchange:

| Queue | Purpose |
| --- | --- |
| `find_available_drivers` | Ask driver service for available drivers for a trip |
| `driver_cmd_trip_request` | Command a specific driver with a trip request |
| `driver_trip_response` | Driver accepts/rejects a trip |
| `notify_driver_no_drivers_found` | No drivers available notification |
| `notify_driver_assign` | Driver assigned to a trip |
| `payment_trip_response` | Payment result for a trip |
| `notify_payment_session_created` | Stripe session created for a trip |
| `payment_success` | Payment confirmed for a trip |
| `dead_letter_queue` | Failed messages routed to the Dead Letter Exchange |

### Getting Started

#### Prerequisites

- Go 1.23+
- Docker
- [Tilt](https://tilt.dev) (`curl -fsSL https://raw.githubusercontent.com/tilt-dev/tilt/master/scripts/install.sh | bash`)
- A local Kubernetes cluster (minikube, kind, k3d, etc.)
- `protoc` + `protoc-gen-go` + `protoc-gen-go-grpc` (only to regenerate protobuf code)

#### 1. Regenerate protobuf code (optional)

Only needed when you change the `.proto` files:

```bash
make generate-proto
```

#### 2. Configure secrets

Create `infra/development/k8s/secrets.yaml` (the template with your keys, it is gitignored):

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: rabbitmq-credentials
type: Opaque
stringData:
  username: "guest"
  password: "guest"
  uri: "amqp://guest:guest@rabbitmq:5672/"
---
apiVersion: v1
kind: Secret
metadata:
  name: stripe-secrets
type: Opaque
stringData:
  stripe-secret-key: "sk_test_..."
  stripe-webhook-key: "whsec_..."
```

#### 3. Start the development environment

```bash
tilt up
```

Tilt compiles the services locally, builds the images, and deploys everything to your cluster with hot reload (`docker_build_with_restart`).

#### Access points

| Component | URL |
| --- | --- |
| Web app | http://localhost:3000 |
| API Gateway | http://localhost:8081 |
| RabbitMQ Management UI | http://localhost:15672 |
| Jaeger UI | http://localhost:16686 |

#### Running services outside Kubernetes (optional)

Each service can also be run directly with Go, as long as RabbitMQ, MongoDB and Jaeger are reachable:

```bash
go run ./services/trip-service/cmd/main.go
go run ./services/payment-service/cmd/main.go
go run ./services/driver-service
go run ./services/api-gateway
```

### Web Frontend

```bash
cd web
npm install
npm run dev
```

The frontend provides separate **rider** and **driver** flows: route selection on a Leaflet map, package (car type) selection, live driver matching via WebSocket, trip overview, and Stripe Checkout payment.

### Creating a New Service

A scaffolding tool generates the clean-architecture layout for a new service:

```bash
go run tools/create_service.go -name=user
```

This creates `services/user-service/` with `cmd/`, `internal/{domain,service,infrastructure}`, `pkg/types/` and a README. Add the new service to the `Tiltfile` following the existing pattern.

### Production

`infra/production/` currently contains Dockerfiles and Kubernetes manifests for the **api-gateway** and **trip-service** only — the rest of the stack is in development.

---

## Español

### ¿Qué es esto?

RideShare es una aplicación de ride-sharing full-stack y orientada a eventos, pensada como plantilla para construir microservicios en Go. Demuestra un flujo completo:

1. Un rider previsualiza un viaje (ruta + paquetes de tarifa) y lo inicia.
2. El trip service encuentra conductores disponibles mediante eventos de RabbitMQ.
3. Los conductores aceptan o rechazan solicitudes de viaje por WebSockets.
4. El rider paga mediante una sesión de Stripe Checkout.
5. Un pago exitoso finaliza el viaje, todo coordinado mediante colas de mensajes.

El backend está escrito en Go, con gRPC para comunicación síncrona entre servicios y RabbitMQ para comunicación asíncrona basada en eventos. El frontend es una aplicación Next.js con mapas en tiempo real.

### Arquitectura

```
                 +-----------------------+
                 |    Next.js Web App    |
                 |      (puerto 3000)    |
                 +-----------+-----------+
                             | HTTP / WebSocket
                 +-----------v-----------+
                 |     API Gateway       |
                 |  REST + WS (puerto 8081)|
                 +---+-------+-------^---+
                     |       |       |
              gRPC   |       |       |  eventos (RabbitMQ)
        +------------+       |       +------------+
        |                    |                    |
+-------v-------+   +--------v-------+   +--------v---------+
| Trip Service  |   | Driver Service |   |  Payment Service |
| gRPC :9093    |   | gRPC :9092     |   |      :9004       |
| MongoDB + OSRM|   | match geohash  |   |     Stripe       |
+-------+-------+   +----------------+   +------------------+
        |  eventos
+-------v-------+
|   RabbitMQ    |   <-- bus de eventos con DLX
+---------------+

+--------+----------+----------+----------+
| Jaeger |  MongoDB |  Stripe  |  OSRM    |
| :16686 |          | (externo) | (externo)|
+--------+----------+----------+----------+
```

Todos los servicios en Go reportan trazas a Jaeger mediante OpenTelemetry.

### Stack Tecnológico

| Capa | Herramientas |
| --- | --- |
| Backend | Go 1.23, gRPC, Protocol Buffers, gorilla/websocket, google/uuid, mmcloughlin/geohash |
| Mensajería | RabbitMQ (AMQP) con Dead Letter Exchange y reintentos |
| Datos | MongoDB (mongo-driver) |
| Pagos | Stripe (stripe-go v81, sesiones Checkout, webhooks) |
| Observabilidad | OpenTelemetry SDK + exportador Jaeger (instrumentación gRPC, HTTP y RabbitMQ) |
| Frontend | Next.js 15, React 19, TypeScript, Tailwind CSS, Leaflet / react-leaflet, Stripe.js, Radix UI |
| Infra | Docker, Kubernetes, Tilt (dev local), Makefile |
| Tooling | Script de scaffolding `tools/create_service.go`, generación de código con `protoc` |

### Estructura del Proyecto

```
.
├── build/                          # Binarios compilados de los servicios (gitignored)
├── infra/
│   ├── development/
│   │   ├── docker/                 # Dockerfiles + scripts de build por servicio
│   │   └── k8s/                    # Manifiestos dev: deployments, config, secrets, jaeger, rabbitmq
│   └── production/
│       ├── docker/                 # Dockerfiles de producción
│       └── k8s/                    # Manifiestos de producción (api-gateway, trip-service)
├── proto/                          # Definiciones .proto (trip, driver)
├── services/
│   ├── api-gateway/                # Punto de entrada HTTP/REST + WebSockets (puerto 8081)
│   ├── driver-service/             # Registro de conductores + matching por geohash (puerto 9092)
│   ├── payment-service/            # Pagos con Stripe (puerto 9004)
│   └── trip-service/               # Rutas, tarifas y ciclo de vida del viaje (puerto 9093)
├── shared/                         # Librería Go compartida
│   ├── contracts/                  # Contratos de payloads AMQP / HTTP / WS
│   ├── db/                         # Helpers del cliente MongoDB
│   ├── env/                        # Helpers de variables de entorno
│   ├── messaging/                  # Wrapper de RabbitMQ: exchanges, colas, DLX, reintentos
│   ├── proto/                      # Código gRPC generado (driver, trip)
│   ├── retry/                      # Utilidades de reintento
│   ├── tracing/                    # Configuración OpenTelemetry + interceptors
│   ├── types/                      # Tipos compartidos
│   └── util/                       # Utilidades
├── tools/
│   └── create_service.go           # Genera un servicio nuevo con clean architecture
├── web/                            # Frontend Next.js (puerto 3000)
├── Makefile                        # Generación de código proto
└── Tiltfile                        # Definición del entorno de desarrollo local
```

Los servicios en Go siguen **Clean Architecture**: `cmd/` (entrypoint), `internal/domain/` (modelos e interfaces de negocio), `internal/service/` (lógica de negocio), `internal/infrastructure/` (repositorios, eventos, gRPC, APIs externas) y `pkg/types/` (tipos públicos compartidos).

### Servicios

| Servicio | Puerto | Responsabilidad |
| --- | --- | --- |
| `web` | 3000 | UI Next.js: mapas de rider/conductores, selector de paquetes, checkout Stripe |
| `api-gateway` | 8081 | Gateway REST + WebSocket; hace proxy gRPC hacia los servicios; receptor del webhook de Stripe |
| `driver-service` | 9092 | gRPC: registro/baja de conductores; matching por geohash; consumidor de solicitudes de viaje |
| `trip-service` | 9093 | gRPC: preview/creación de viajes; obtención de rutas (OSRM); estimación de tarifas; ciclo de vida + publisher de eventos |
| `payment-service` | 9004 | Sesiones Stripe Checkout y manejo de webhooks mediante consumidores de RabbitMQ |

**Endpoints del API Gateway:**

- `POST /trip/preview` — previsualiza ruta y paquetes de tarifa
- `POST /trip/start` — inicia un viaje con una tarifa seleccionada
- `WS /ws/drivers` — stream en tiempo real para conductores (solicitudes y asignaciones)
- `WS /ws/riders` — stream en tiempo real para riders (actualizaciones de matching)
- `POST /webhook/stripe` — webhook de pagos de Stripe

### Eventos RabbitMQ

Flujo orientado a eventos entre servicios sobre el exchange `trip`:

| Cola | Propósito |
| --- | --- |
| `find_available_drivers` | Consulta al driver service por conductores disponibles para un viaje |
| `driver_cmd_trip_request` | Comando a un conductor específico con una solicitud de viaje |
| `driver_trip_response` | El conductor acepta/rechaza un viaje |
| `notify_driver_no_drivers_found` | Notificación de que no hay conductores disponibles |
| `notify_driver_assign` | Conductor asignado a un viaje |
| `payment_trip_response` | Resultado del pago de un viaje |
| `notify_payment_session_created` | Sesión de Stripe creada para un viaje |
| `payment_success` | Pago confirmado para un viaje |
| `dead_letter_queue` | Mensajes fallidos enrutados al Dead Letter Exchange |

### Cómo Montarlo

#### Requisitos previos

- Go 1.23+
- Docker
- [Tilt](https://tilt.dev) (`curl -fsSL https://raw.githubusercontent.com/tilt-dev/tilt/master/scripts/install.sh | bash`)
- Un clúster Kubernetes local (minikube, kind, k3d, etc.)
- `protoc` + `protoc-gen-go` + `protoc-gen-go-grpc` (solo para regenerar el código de protobuf)

#### 1. Regenerar el código protobuf (opcional)

Solo es necesario si modificas los archivos `.proto`:

```bash
make generate-proto
```

#### 2. Configurar los secrets

Crea `infra/development/k8s/secrets.yaml` (la plantilla con tus keys, está en gitignore):

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: rabbitmq-credentials
type: Opaque
stringData:
  username: "guest"
  password: "guest"
  uri: "amqp://guest:guest@rabbitmq:5672/"
---
apiVersion: v1
kind: Secret
metadata:
  name: stripe-secrets
type: Opaque
stringData:
  stripe-secret-key: "sk_test_..."
  stripe-webhook-key: "whsec_..."
```

#### 3. Levantar el entorno de desarrollo

```bash
tilt up
```

Tilt compila los servicios localmente, construye las imágenes y despliega todo en tu clúster con recarga en caliente (`docker_build_with_restart`).

#### Puntos de acceso

| Componente | URL |
| --- | --- |
| Web app | http://localhost:3000 |
| API Gateway | http://localhost:8081 |
| RabbitMQ Management UI | http://localhost:15672 |
| Jaeger UI | http://localhost:16686 |

#### Ejecutar los servicios fuera de Kubernetes (opcional)

Cada servicio también puede ejecutarse directamente con Go, siempre que RabbitMQ, MongoDB y Jaeger sean alcanzables:

```bash
go run ./services/trip-service/cmd/main.go
go run ./services/payment-service/cmd/main.go
go run ./services/driver-service
go run ./services/api-gateway
```

### Frontend Web

```bash
cd web
npm install
npm run dev
```

El frontend ofrece flujos separados para **rider** y **conductor**: selección de ruta en un mapa Leaflet, selección de paquete (tipo de vehículo), matching de conductores en vivo vía WebSocket, resumen del viaje y pago con Stripe Checkout.

### Crear un Nuevo Servicio

Una herramienta de scaffolding genera el layout de clean architecture para un servicio nuevo:

```bash
go run tools/create_service.go -name=user
```

Esto crea `services/user-service/` con `cmd/`, `internal/{domain,service,infrastructure}`, `pkg/types/` y un README. Añade el servicio nuevo al `Tiltfile` siguiendo el patrón existente.

### Producción

`infra/production/` contiene actualmente Dockerfiles y manifiestos de Kubernetes solo para el **api-gateway** y el **trip-service** — el resto del stack está en desarrollo.