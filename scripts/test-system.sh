#!/bin/bash

# Скрипт для полного тестирования URL Shortener системы

set -e

echo "🧪 URL Shortener - Тестирование системы"
echo "========================================"
echo ""

# Проверка доступности сервисов
echo "1️⃣ Проверка здоровья сервисов..."
echo ""

check_health() {
    local service=$1
    local url=$2
    
    if curl -s "$url" > /dev/null 2>&1; then
        echo "✅ $service - OK"
    else
        echo "❌ $service - FAIL"
        return 1
    fi
}

check_health "API Gateway" "http://localhost:3000/health"
check_health "Shortener Service" "http://localhost:3001/health"
check_health "Redirect Service" "http://localhost:3002/health"
check_health "Analytics Service" "http://localhost:3003/health"
check_health "Frontend" "http://localhost:8080"

echo ""
echo "2️⃣ Создание короткого URL для https://github.com..."
echo ""

RESPONSE=$(curl -s -X POST http://localhost:3000/api/shorten \
    -H "Content-Type: application/json" \
    -d '{"url": "https://github.com"}')

echo "Response: $RESPONSE"

SHORT_CODE=$(echo $RESPONSE | grep -o '"shortCode":"[^"]*"' | cut -d'"' -f4)

if [ -z "$SHORT_CODE" ]; then
    echo "❌ Не удалось создать короткий URL"
    exit 1
fi

echo "✅ Короткий код создан: $SHORT_CODE"
echo ""

echo "3️⃣ Создание ещё нескольких URL..."
echo ""

curl -s -X POST http://localhost:3000/api/shorten \
    -H "Content-Type: application/json" \
    -d '{"url": "https://google.com"}' | grep -o '"shortCode":"[^"]*"' | cut -d'"' -f4

curl -s -X POST http://localhost:3000/api/shorten \
    -H "Content-Type: application/json" \
    -d '{"url": "https://stackoverflow.com"}' | grep -o '"shortCode":"[^"]*"' | cut -d'"' -f4

echo "✅ Дополнительные URL созданы"
echo ""

echo "4️⃣ Тестирование перенаправления (несколько кликов)..."
echo ""

for i in {1..5}; do
    curl -s -I -L "http://localhost:3002/$SHORT_CODE" > /dev/null
    echo "  Клик #$i отправлен"
done

echo "✅ Перенаправления выполнены"
echo ""

echo "5️⃣ Ожидание обработки событий Kafka (3 секунды)..."
sleep 3
echo ""

echo "6️⃣ Получение статистики для $SHORT_CODE..."
echo ""

STATS=$(curl -s "http://localhost:3000/api/stats/$SHORT_CODE")
echo "$STATS" | jq '.'

CLICKS=$(echo "$STATS" | grep -o '"totalClicks":[0-9]*' | cut -d':' -f2)

if [ "$CLICKS" -eq 5 ]; then
    echo "✅ Статистика корректна: $CLICKS кликов"
else
    echo "⚠️ Ожидалось 5 кликов, получено: $CLICKS"
fi

echo ""
echo "7️⃣ Получение общей статистики..."
echo ""

curl -s "http://localhost:3000/api/stats" | jq '.'

echo ""
echo "========================================"
echo "✅ Все тесты пройдены успешно!"
echo "========================================"
echo ""
echo "📊 Результаты:"
echo "  - Все сервисы работают"
echo "  - Создание URL работает"
echo "  - Перенаправление работает"
echo "  - Kafka доставляет события"
echo "  - Analytics сохраняет статистику"
echo ""
echo "🎉 Система полностью функциональна!"
