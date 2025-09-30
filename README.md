# URL Shortener - Микросервисная архитектура на Go

Демонстрационный проект микросервисной архитектуры с использованием Go, Kafka, Redis и MongoDB.

## Обзор

Проект реализует сервис сокращения URL с полноценной микросервисной архитектурой, демонстрирующий:

- **Event-Driven Architecture** - асинхронное взаимодействие через Kafka
- **API Gateway паттерн** - единая точка входа
- **Database per Service** - каждый сервис имеет свою БД
- **Distributed Tracing** - OpenTelemetry + Jaeger
- **CI/CD** - GitHub Actions с динамической матрицей сборки

## Архитектура

### Микросервисы

1. **api-gateway** (:3000) - точка входа для всех API запросов, маршрутизация
2. **shortener-service** (:3001) - создание коротких URL, работа с Redis
3. **redirect-service** (:3002) - перенаправление по коротким URL, публикация событий в Kafka
4. **analytics-service** (:3003) - обработка событий из Kafka, сохранение статистики в MongoDB
5. **frontend** (:8080) - веб-интерфейс на HTML/CSS/JS + Nginx

### Инфраструктура

- **Redis** - хранилище URL маппинга (in-memory)
- **MongoDB** - база данных для аналитики
- **Kafka + Zookeeper** - шина сообщений для асинхронного взаимодействия
- **Jaeger** - distributed tracing для мониторинга

### Поток данных

```
Создание URL:
  User → api-gateway → shortener-service → Redis

Переход по ссылке:
  User → redirect-service → Redis (получить URL)
                         → Kafka (событие клика)
                         → analytics-service → MongoDB

Просмотр статистики:
  User → api-gateway → analytics-service → MongoDB
```

## Быстрый старт

### Требования

- Docker
- Docker Compose

### Запуск

```bash
# С использованием Make
make up-build

# Или напрямую через Docker Compose
docker-compose up --build
```

### Проверка

Откройте браузер: http://localhost:8080

Jaeger UI (трейсинг): http://localhost:16686

## API

### Создать короткий URL

```bash
curl -X POST http://localhost:3000/api/shorten \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com"}'
```

Ответ:
```json
{
  "shortCode": "abc123",
  "shortUrl": "http://localhost:3002/abc123",
  "originalUrl": "https://example.com"
}
```

### Получить статистику

```bash
# Статистика конкретного URL
curl http://localhost:3000/api/stats/abc123

# Вся статистика
curl http://localhost:3000/api/stats
```

### Перейти по короткой ссылке

```bash
curl -L http://localhost:3002/abc123
```

## Основные команды

```bash
# Посмотреть все команды
make help

# Запустить систему
make up-build

# Проверить здоровье сервисов
make health

# Полный тест системы
make test

# Посмотреть логи
make logs

# Остановить систему
make down

# Удалить все контейнеры и данные
make clean
```

## CI/CD

Проект использует GitHub Actions с автоматическим определением изменённых сервисов:

- **build-pr.yml** - собирает только изменённые сервисы при создании PR
- **build-main.yml** - собирает изменённые сервисы после merge в main
- **build-all.yml** - ручная сборка всех сервисов

Образы публикуются в GitHub Container Registry:
```
ghcr.io/itcaat/url-shortener-demo/{service}:latest
```

При изменении `pkg/` пересобираются все Go-сервисы.

## Структура проекта

```
.
├── api-gateway/              # API Gateway сервис
├── shortener-service/        # Сервис создания коротких URL
├── redirect-service/         # Сервис перенаправления
├── analytics-service/        # Сервис аналитики
├── frontend/                 # Веб-интерфейс
├── pkg/tracing/             # Общая библиотека для трейсинга
├── docker-compose.yml       # Оркестрация сервисов
├── docker-compose.debug.yml # Конфигурация с Jaeger
└── Makefile                 # Команды для управления
```

## Технологии

| Компонент | Технология | Версия |
|-----------|------------|--------|
| Backend | Go | 1.21 |
| Message Broker | Apache Kafka | 7.5.0 |
| Cache | Redis | 7 |
| Database | MongoDB | 7 |
| Tracing | Jaeger | 1.52 |
| Frontend | Nginx | Alpine |
| Orchestration | Docker Compose | Latest |

## Отладка

### Логи сервисов

```bash
# Все сервисы
docker-compose logs -f

# Конкретный сервис
docker-compose logs -f api-gateway
```

### Kafka

```bash
# Подключиться к Kafka
docker exec -it url-shortener-kafka bash

# Читать сообщения из топика
kafka-console-consumer --bootstrap-server localhost:9092 \
  --topic url-clicks --from-beginning
```

### MongoDB

```bash
# Подключиться к MongoDB
docker exec -it url-shortener-mongodb mongosh

use analytics
db.clicks.find().pretty()
```

### Redis

```bash
# Подключиться к Redis
docker exec -it url-shortener-redis redis-cli

KEYS url:*
GET url:abc123
```

## Масштабирование

```bash
# Запустить несколько инстансов analytics-service
docker-compose up -d --scale analytics-service=3

# Kafka автоматически распределит партиции между consumer'ами
```

## Мониторинг

### Health Checks

```bash
curl http://localhost:3000/health  # API Gateway
curl http://localhost:3001/health  # Shortener Service
curl http://localhost:3002/health  # Redirect Service
curl http://localhost:3003/health  # Analytics Service
```

### Jaeger Tracing

Откройте http://localhost:16686 для просмотра распределённых трейсов запросов через все микросервисы.

## Идеи для развития

- Аутентификация (JWT)
- Rate limiting
- TTL для коротких URL
- Custom aliases
- QR code генерация
- Prometheus + Grafana
- Kubernetes deployment

## Лицензия

MIT

---

Демонстрационный проект для изучения микросервисной архитектуры
