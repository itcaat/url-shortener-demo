.PHONY: help build up down logs clean test

help: ## Показать помощь
	@echo "URL Shortener - Микросервисная архитектура"
	@echo ""
	@echo "Доступные команды:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Собрать все Docker образы
	docker-compose build

up: ## Запустить все сервисы
	docker-compose up -d
	@echo ""
	@echo "✅ Все сервисы запущены!"
	@echo ""
	@echo "🌐 Frontend:        http://localhost:8080"
	@echo "🔌 API Gateway:    http://localhost:3000"
	@echo "📊 Redirect:       http://localhost:3002"
	@echo ""

up-build: ## Собрать и запустить все сервисы
	docker-compose up -d --build
	@echo ""
	@echo "✅ Все сервисы собраны и запущены!"
	@echo ""
	@echo "🌐 Frontend:        http://localhost:8080"
	@echo "🔌 API Gateway:    http://localhost:3000"
	@echo "📊 Redirect:       http://localhost:3002"
	@echo ""

down: ## Остановить все сервисы
	docker-compose down

logs: ## Показать логи всех сервисов
	docker-compose logs -f

logs-api: ## Показать логи API Gateway
	docker-compose logs -f api-gateway

logs-shortener: ## Показать логи Shortener Service
	docker-compose logs -f shortener-service

logs-redirect: ## Показать логи Redirect Service
	docker-compose logs -f redirect-service

logs-analytics: ## Показать логи Analytics Service
	docker-compose logs -f analytics-service

logs-kafka: ## Показать логи Kafka
	docker-compose logs -f kafka

status: ## Показать статус сервисов
	docker-compose ps

health: ## Проверить здоровье всех сервисов
	@echo "Проверка здоровья сервисов..."
	@echo ""
	@echo "API Gateway:"
	@curl -s http://localhost:3000/health | jq '.' || echo "❌ Недоступен"
	@echo ""
	@echo "Shortener Service:"
	@curl -s http://localhost:3001/health | jq '.' || echo "❌ Недоступен"
	@echo ""
	@echo "Redirect Service:"
	@curl -s http://localhost:3002/health | jq '.' || echo "❌ Недоступен"
	@echo ""
	@echo "Analytics Service:"
	@curl -s http://localhost:3003/health | jq '.' || echo "❌ Недоступен"
	@echo ""

test: ## Запустить полный тестовый сценарий
	@bash scripts/test-system.sh

test-simple: ## Запустить простой тестовый сценарий
	@echo "🧪 Запуск тестового сценария..."
	@echo ""
	@echo "1️⃣ Создание короткого URL для https://github.com"
	@curl -s -X POST http://localhost:3000/api/shorten \
		-H "Content-Type: application/json" \
		-d '{"url": "https://github.com"}' | jq '.'
	@echo ""
	@echo "2️⃣ Создание короткого URL для https://google.com"
	@curl -s -X POST http://localhost:3000/api/shorten \
		-H "Content-Type: application/json" \
		-d '{"url": "https://google.com"}' | jq '.'
	@echo ""
	@echo "3️⃣ Получение всей статистики"
	@sleep 2
	@curl -s http://localhost:3000/api/stats | jq '.'
	@echo ""

clean: ## Удалить все контейнеры и volumes
	docker-compose down -v
	@echo "✅ Все удалено!"

restart: down up ## Перезапустить все сервисы

scale-analytics: ## Масштабировать analytics-service (3 инстанса)
	docker-compose up -d --scale analytics-service=3
	@echo "✅ Analytics service масштабирован до 3 инстансов"

redis-cli: ## Подключиться к Redis CLI
	docker exec -it url-shortener-redis redis-cli

mongo-cli: ## Подключиться к MongoDB CLI
	docker exec -it url-shortener-mongodb mongosh analytics

kafka-topics: ## Показать Kafka топики
	docker exec -it url-shortener-kafka kafka-topics --bootstrap-server localhost:9092 --list

kafka-consume: ## Читать сообщения из Kafka топика url-clicks
	docker exec -it url-shortener-kafka kafka-console-consumer \
		--bootstrap-server localhost:9092 \
		--topic url-clicks \
		--from-beginning

jaeger: ## Открыть Jaeger UI
	@echo "🔍 Opening Jaeger UI at http://localhost:16686"
	@open http://localhost:16686 2>/dev/null || xdg-open http://localhost:16686 2>/dev/null || echo "Please open http://localhost:16686 in your browser"

dev: ## Режим разработки (логи в реальном времени)
	docker-compose up --build
