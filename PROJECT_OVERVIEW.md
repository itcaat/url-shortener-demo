# 📋 Обзор проекта URL Shortener

## 🎯 Цель проекта

Демонстрация микросервисной архитектуры на практическом примере с использованием современных технологий и best practices.

## 📁 Структура файлов

```
github-actions-matrix-example/
│
├── 📄 README.md                 # Основная документация
├── 📄 QUICKSTART.md             # Быстрый старт
├── 📄 ARCHITECTURE.md           # Детальное описание архитектуры
├── 📄 PROJECT_OVERVIEW.md       # Этот файл
│
├── 🐳 docker-compose.yml        # Оркестрация всех сервисов
├── 📝 Makefile                  # Удобные команды
├── 🔧 .gitignore               # Git ignore файл
├── 🔧 .dockerignore            # Docker ignore файл
│
├── 📁 api-gateway/              # Сервис: API Gateway
│   ├── main.go                  # Основной код (Go)
│   ├── go.mod                   # Go dependencies
│   └── Dockerfile               # Docker образ
│
├── 📁 shortener-service/        # Сервис: Создание коротких URL
│   ├── main.go                  # Основной код (Go + Redis)
│   ├── go.mod                   # Go dependencies
│   └── Dockerfile               # Docker образ
│
├── 📁 redirect-service/         # Сервис: Перенаправление
│   ├── main.go                  # Основной код (Go + Redis + Kafka Producer)
│   ├── go.mod                   # Go dependencies
│   └── Dockerfile               # Docker образ
│
├── 📁 analytics-service/        # Сервис: Аналитика
│   ├── main.go                  # Основной код (Go + MongoDB + Kafka Consumer)
│   ├── go.mod                   # Go dependencies
│   └── Dockerfile               # Docker образ
│
├── 📁 frontend/                 # Веб-интерфейс
│   ├── index.html               # HTML страница
│   └── Dockerfile               # Nginx образ
│
└── 📁 scripts/                  # Вспомогательные скрипты
    └── test-system.sh           # Скрипт тестирования
```

## 🔢 Статистика проекта

### Микросервисы
- **Всего:** 4 Go-сервиса
- **Языки:** Go 1.21
- **Строк кода (примерно):**
  - api-gateway: ~200 строк
  - shortener-service: ~230 строк
  - redirect-service: ~220 строк
  - analytics-service: ~270 строк
  - **Итого:** ~920 строк Go кода

### Инфраструктура
- **Контейнеры:** 9 (4 сервиса + frontend + redis + mongodb + kafka + zookeeper)
- **Базы данных:** 2 (Redis, MongoDB)
- **Message Broker:** 1 (Kafka + Zookeeper)
- **Веб-сервер:** 1 (Nginx)

### Порты
```
3000 - API Gateway
3001 - Shortener Service
3002 - Redirect Service
3003 - Analytics Service
6379 - Redis
9092 - Kafka
27017 - MongoDB
8080 - Frontend (Nginx)
```

## 🏗️ Архитектурные паттерны

| Паттерн | Применение | Где |
|---------|-----------|-----|
| **API Gateway** | Единая точка входа | api-gateway |
| **Database per Service** | Каждый сервис имеет свою БД | shortener → Redis, analytics → MongoDB |
| **Event-Driven** | Асинхронное взаимодействие | redirect → Kafka → analytics |
| **CQRS** | Разделение чтения/записи | Разные сервисы для создания и статистики |
| **Service Discovery** | Поиск сервисов | Docker DNS |
| **Health Check** | Мониторинг здоровья | /health endpoints |
| **Circuit Breaker** | (можно добавить) | Будущее улучшение |

## 🔄 Жизненный цикл запроса

### 1. Создание короткого URL

```
┌─────────┐     ┌─────────────┐     ┌──────────────┐     ┌───────┐
│ Browser │────▶│ API Gateway │────▶│  Shortener   │────▶│ Redis │
│         │     │  :3000      │     │   :3001      │     │:6379  │
└─────────┘     └─────────────┘     └──────────────┘     └───────┘
     ▲                                       │
     └───────────────────────────────────────┘
           Возврат shortCode: abc123
```

**Время:** ~10-50ms

### 2. Переход по короткому URL

```
┌─────────┐     ┌──────────┐     ┌───────┐
│ Browser │────▶│ Redirect │────▶│ Redis │
│         │     │  :3002   │     │:6379  │
└─────────┘     └──────────┘     └───────┘
     ▲               │
     │               ▼
     │          ┌────────┐     ┌───────────┐     ┌─────────┐
     │          │ Kafka  │────▶│ Analytics │────▶│ MongoDB │
     │          │:9092   │     │  :3003    │     │ :27017  │
     │          └────────┘     └───────────┘     └─────────┘
     │
     └─── HTTP 302 Redirect
```

**Время пользователя:** ~5-20ms  
**Обработка аналитики:** асинхронно

### 3. Получение статистики

```
┌─────────┐     ┌─────────────┐     ┌───────────┐     ┌─────────┐
│ Browser │────▶│ API Gateway │────▶│ Analytics │────▶│ MongoDB │
│         │     │  :3000      │     │  :3003    │     │ :27017  │
└─────────┘     └─────────────┘     └───────────┘     └─────────┘
     ▲                                       │
     └───────────────────────────────────────┘
              Возврат статистики
```

**Время:** ~50-200ms

## 📊 Поток данных через Kafka

```
Event: Click on short URL
│
├─ Producer: redirect-service
│  └─ Topic: url-clicks
│     └─ Message: {shortCode, timestamp, userAgent, ip}
│
└─ Consumer: analytics-service
   └─ Consumer Group: analytics-consumer-group
      └─ Action: Save to MongoDB
```

## 🚀 Сценарии использования

### Сценарий 1: Маркетинговая кампания
- Создать короткий URL для лендинга
- Распространить в соц. сетях
- Отслеживать количество переходов в реальном времени

### Сценарий 2: A/B тестирование
- Создать несколько коротких URL для разных версий
- Сравнить статистику переходов

### Сценарий 3: QR коды (будущее)
- Генерировать QR код для короткого URL
- Использовать на печатных материалах
- Отслеживать офлайн-конверсию

## 🎓 Учебная ценность

### Что можно изучить:

#### 1. Микросервисы
- ✅ Разделение на независимые сервисы
- ✅ Взаимодействие между сервисами
- ✅ Изоляция сбоев
- ✅ Независимое масштабирование

#### 2. Go Programming
- ✅ HTTP серверы (net/http)
- ✅ REST API (Gorilla Mux)
- ✅ Работа с Redis (go-redis)
- ✅ Работа с MongoDB (mongo-driver)
- ✅ Kafka Producer/Consumer (segmentio/kafka-go)
- ✅ Горутины и конкурентность

#### 3. Message Brokers
- ✅ Apache Kafka
- ✅ Topics и Partitions
- ✅ Consumer Groups
- ✅ Producer/Consumer паттерн
- ✅ Асинхронная обработка

#### 4. Базы данных
- ✅ Redis - key-value store
- ✅ MongoDB - document database
- ✅ Выбор правильной БД под задачу

#### 5. DevOps
- ✅ Docker containerization
- ✅ Docker Compose orchestration
- ✅ Multi-stage builds
- ✅ Health checks
- ✅ Logging

#### 6. Архитектурные концепции
- ✅ API Gateway pattern
- ✅ Event-Driven Architecture
- ✅ CQRS
- ✅ Service Discovery
- ✅ Polyglot Persistence

## 🔧 Команды для разработки

```bash
# Начало работы
make help              # Все доступные команды
make up-build          # Запустить систему
make health            # Проверить здоровье
make test              # Полный тест

# Разработка
make logs              # Все логи
make logs-api          # Логи API Gateway
make logs-kafka        # Логи Kafka

# Отладка
make redis-cli         # Redis консоль
make mongo-cli         # MongoDB консоль
make kafka-topics      # Список топиков
make kafka-consume     # Читать сообщения

# Масштабирование
make scale-analytics   # 3 инстанса analytics

# Очистка
make down              # Остановить
make clean             # Удалить всё
make restart           # Перезапустить
```

## 📈 Метрики производительности

### Throughput (примерно, локально)
- **Создание URL:** ~1000 req/s
- **Redirect:** ~5000 req/s
- **Stats API:** ~500 req/s

### Latency (50th percentile)
- **Создание URL:** ~10ms
- **Redirect:** ~5ms
- **Stats API:** ~50ms

### Kafka
- **Throughput:** 10000+ msg/s
- **Latency:** <10ms

*Реальные цифры зависят от железа*

## 🛡️ Production-ready чеклист

- [ ] Аутентификация (JWT)
- [ ] Rate limiting
- [ ] Input validation
- [ ] HTTPS/TLS
- [ ] Secrets management
- [ ] Monitoring (Prometheus)
- [ ] Logging (ELK/Loki)
- [ ] Tracing (Jaeger)
- [ ] CI/CD pipeline
- [ ] Kubernetes deployment
- [ ] Автотесты
- [ ] Documentation
- [ ] Error handling
- [ ] Graceful shutdown
- [ ] Circuit breaker
- [ ] Retry logic
- [ ] Backups
- [ ] Disaster recovery

## 🎯 Roadmap

### v1.0 (Текущая версия)
- ✅ Базовая функциональность
- ✅ Микросервисная архитектура
- ✅ Kafka интеграция
- ✅ Docker Compose

### v2.0 (Планируется)
- [ ] Kubernetes deployment
- [ ] Helm charts
- [ ] GitHub Actions CI/CD
- [ ] Unit/Integration тесты
- [ ] Prometheus + Grafana
- [ ] Custom aliases
- [ ] TTL для URL
- [ ] QR коды

### v3.0 (Идеи)
- [ ] Service Mesh (Istio)
- [ ] GraphQL API
- [ ] WebSocket для real-time stats
- [ ] Multi-tenancy
- [ ] API versioning
- [ ] Rate limiting per user
- [ ] Analytics dashboard

## 🤝 Contributing

Идеи для вклада:
1. Добавить unit тесты
2. Реализовать TTL для URL
3. Добавить custom aliases
4. Создать Kubernetes манифесты
5. Добавить Prometheus метрики
6. Реализовать QR генерацию
7. Добавить API документацию (Swagger)
8. Улучшить UI

## 📚 Полезные ресурсы

### Микросервисы
- [Microservices.io](https://microservices.io/)
- [Building Microservices by Sam Newman](https://samnewman.io/books/building_microservices/)

### Go
- [Go by Example](https://gobyexample.com/)
- [Effective Go](https://golang.org/doc/effective_go)

### Kafka
- [Kafka Documentation](https://kafka.apache.org/documentation/)
- [Confluent Tutorials](https://kafka-tutorials.confluent.io/)

### Docker
- [Docker Documentation](https://docs.docker.com/)
- [Docker Compose Reference](https://docs.docker.com/compose/)

## 📞 Поддержка

Если есть вопросы:
1. Прочитайте [QUICKSTART.md](./QUICKSTART.md)
2. Изучите [ARCHITECTURE.md](./ARCHITECTURE.md)
3. Посмотрите логи: `make logs`
4. Создайте Issue в GitHub

---

**Автор:** Демонстрационный проект для обучения  
**Лицензия:** MIT  
**Версия:** 1.0.0  

Удачи в изучении микросервисов! 🚀
