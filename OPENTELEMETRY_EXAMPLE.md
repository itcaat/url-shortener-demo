# 🔧 OpenTelemetry Integration Example

Этот документ показывает как добавить OpenTelemetry трейсинг в Go микросервисы для отправки данных в Jaeger.

## 📋 Текущий статус

✅ **Jaeger запущен** и доступен на http://localhost:16686  
✅ **Все сервисы настроены** с переменными окружения Jaeger  
⚠️ **Код трейсинга** - нужно добавить OpenTelemetry библиотеки  

## 🎯 Что нужно сделать

Чтобы увидеть трейсы в Jaeger UI, нужно:
1. Добавить OpenTelemetry зависимости в `go.mod`
2. Инициализировать tracer в `main()`
3. Добавить HTTP middleware для автоматического трейсинга
4. (Опционально) Добавить custom spans для деталей

## 📝 Пример для api-gateway

### Шаг 1: Обновить go.mod

```go
module api-gateway

go 1.21

require (
	github.com/gorilla/mux v1.8.1
	github.com/rs/cors v1.10.1
	go.opentelemetry.io/otel v1.21.0
	go.opentelemetry.io/otel/exporters/jaeger v1.17.0
	go.opentelemetry.io/otel/sdk v1.21.0
	go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux v0.46.1
)
```

### Шаг 2: Добавить функцию инициализации трейсинга

Создайте файл `tracing.go`:

```go
package main

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

func initTracer(serviceName string) (*tracesdk.TracerProvider, error) {
	jaegerHost := getEnv("JAEGER_AGENT_HOST", "localhost")
	jaegerPort := getEnv("JAEGER_AGENT_PORT", "6831")

	exp, err := jaeger.New(
		jaeger.WithAgentEndpoint(
			jaeger.WithAgentHost(jaegerHost),
			jaeger.WithAgentPort(jaegerPort),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Jaeger exporter: %w", err)
	}

	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exp),
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
		)),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp, nil
}
```

### Шаг 3: Обновить main.go

```go
package main

import (
	"context"
	"log"
	"net/http"
	
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
)

func main() {
	// Инициализация трейсинга
	tp, err := initTracer("api-gateway")
	if err != nil {
		log.Printf("Failed to initialize tracer: %v", err)
	} else {
		defer func() {
			if err := tp.Shutdown(context.Background()); err != nil {
				log.Printf("Error shutting down tracer provider: %v", err)
			}
		}()
		log.Println("Jaeger tracing initialized")
	}

	router := mux.NewRouter()
	
	// ВАЖНО: Добавить OpenTelemetry middleware
	router.Use(otelmux.Middleware("api-gateway"))

	// Ваши routes...
	router.HandleFunc("/health", healthHandler).Methods("GET")
	router.HandleFunc("/api/shorten", shortenHandler).Methods("POST")
	// ... остальные routes

	handler := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	}).Handler(router)

	port := getEnv("PORT", "3000")
	log.Printf("[API Gateway] Server starting on port %s\n", port)
	
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal(err)
	}
}
```

### Шаг 4: Добавить custom spans (опционально)

Для более детальных трейсов:

```go
import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func shortenHandler(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("api-gateway")
	ctx, span := tracer.Start(r.Context(), "proxy_to_shortener")
	defer span.End()

	// Добавить атрибуты
	span.SetAttributes(
		attribute.String("http.method", "POST"),
		attribute.String("target.service", "shortener-service"),
	)

	// Ваш код...
	
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}
```

## 🔄 Применить для всех сервисов

Повторите те же шаги для:
- `shortener-service`
- `redirect-service`
- `analytics-service`

## 📊 Что вы увидите в Jaeger

После добавления OpenTelemetry:

### 1. Service Graph
```
api-gateway → shortener-service → Redis
```

### 2. Trace Details
```
Trace ID: abc123...
Duration: 15ms

├─ api-gateway (5ms)
│  └─ POST /api/shorten
│
└─ shortener-service (10ms)
   ├─ HTTP POST handler (2ms)
   └─ Redis SET (8ms)
```

### 3. Metrics
- Latency: P50, P95, P99
- Request rate
- Error rate

## 🎯 Best Practices

### 1. Span Names
```go
// ✅ Хорошо
span := tracer.Start(ctx, "http.POST /api/shorten")

// ❌ Плохо
span := tracer.Start(ctx, "handler")
```

### 2. Атрибуты
```go
span.SetAttributes(
	attribute.String("http.method", r.Method),
	attribute.String("http.url", r.URL.String()),
	attribute.Int("http.status_code", statusCode),
	attribute.String("db.system", "redis"),
)
```

### 3. Ошибки
```go
if err != nil {
	span.RecordError(err)
	span.SetStatus(codes.Error, "Failed to save URL")
	return err
}
```

### 4. Propagation
```go
// Передать контекст в HTTP запрос
req, _ := http.NewRequestWithContext(ctx, "POST", url, body)

// OpenTelemetry автоматически добавит headers:
// traceparent: 00-{trace-id}-{span-id}-01
```

## 🔧 Трейсинг Kafka

Для асинхронных операций через Kafka:

```go
import (
	"go.opentelemetry.io/otel/propagation"
)

// Producer (redirect-service)
func publishToKafka(ctx context.Context, event ClickEvent) {
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	
	// Добавить carrier в Kafka headers
	headers := []kafka.Header{}
	for k, v := range carrier {
		headers = append(headers, kafka.Header{
			Key:   k,
			Value: []byte(v),
		})
	}
	
	message := kafka.Message{
		Key:     []byte(event.ShortCode),
		Value:   jsonData,
		Headers: headers,  // <- передаем trace context
	}
}

// Consumer (analytics-service)
func consumeFromKafka(msg kafka.Message) {
	carrier := propagation.MapCarrier{}
	for _, h := range msg.Headers {
		carrier[h.Key] = string(h.Value)
	}
	
	ctx := otel.GetTextMapPropagator().Extract(context.Background(), carrier)
	
	tracer := otel.Tracer("analytics-service")
	ctx, span := tracer.Start(ctx, "kafka.consume url-clicks")
	defer span.End()
	
	// Обработка сообщения...
}
```

## 🚀 Запуск с трейсингом

```bash
# 1. Обновить зависимости
cd api-gateway && go mod tidy

# 2. Пересобрать образы
docker-compose build

# 3. Перезапустить
docker-compose down
docker-compose up -d

# 4. Создать запросы
curl -X POST http://localhost:3000/api/shorten \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com"}'

# 5. Открыть Jaeger UI
open http://localhost:16686

# 6. Выбрать сервис "api-gateway" и нажать "Find Traces"
```

## 📚 Дополнительные ресурсы

- [OpenTelemetry Go](https://opentelemetry.io/docs/instrumentation/go/)
- [Jaeger Documentation](https://www.jaegertracing.io/docs/)
- [OpenTelemetry Best Practices](https://opentelemetry.io/docs/concepts/signals/traces/)
- [Context Propagation](https://opentelemetry.io/docs/instrumentation/go/manual/#propagators-and-context)

## 💡 Tip

Для быстрого тестирования можно добавить трейсинг только в один сервис (например, api-gateway) и посмотреть как это работает, а потом добавить в остальные.

---

**Статус:** Jaeger готов, код OpenTelemetry - опциональное расширение  
**Сложность:** Средняя (требует понимания Go и OpenTelemetry)  
**Время:** ~2-3 часа для всех сервисов  

Удачи с трейсингом! 🔍
