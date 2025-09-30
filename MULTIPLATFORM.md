# 🌐 Multi-Platform Build Guide

Инструкции по сборке Docker образов для различных платформ (AMD64 и ARM64).

## 🎯 Зачем нужно?

- ✅ **Apple Silicon (M1/M2/M3)** - требуют ARM64 образы
- ✅ **Intel Mac / Linux** - работают с AMD64 образами
- ✅ **Серверы** - обычно AMD64, но есть ARM64 (AWS Graviton, etc.)

## 🚀 GitHub Actions

Все workflows уже настроены для multi-platform сборки:

```yaml
platforms: linux/amd64,linux/arm64
```

**Собираются автоматически:**
- ✅ При создании PR (workflow: `build-pr.yml`)
- ✅ При push в main (workflow: `build-main.yml`)
- ✅ При ручном запуске (workflow: `build-all.yml`)

## 💻 Локальная разработка

### Способ 1: docker-compose (рекомендуется)

`docker-compose` автоматически выбирает правильную платформу:

```bash
# Обычная сборка и запуск
docker-compose up --build

# Docker Compose автоматически использует вашу платформу
# Mac M1/M2/M3 → ARM64
# Intel Mac/Linux → AMD64
```

### Способ 2: Docker Buildx (для multi-platform)

Если нужно собрать образы для нескольких платформ:

#### 1. Настройка Buildx (один раз)

```bash
# Создать buildx builder
docker buildx create --name multiplatform --use

# Проверить
docker buildx inspect multiplatform --bootstrap
```

#### 2. Сборка multi-platform образа

```bash
# Пример: api-gateway
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t myrepo/api-gateway:latest \
  -f api-gateway/Dockerfile \
  --push \
  .

# Или загрузить локально (только одна платформа за раз)
docker buildx build \
  --platform linux/arm64 \
  -t api-gateway:local \
  -f api-gateway/Dockerfile \
  --load \
  .
```

#### 3. Скрипт для сборки всех сервисов

```bash
#!/bin/bash
# build-multiplatform.sh

SERVICES=("api-gateway" "shortener-service" "redirect-service" "analytics-service" "frontend")
PLATFORMS="linux/amd64,linux/arm64"
TAG="local"

for service in "${SERVICES[@]}"; do
  echo "🔨 Building $service..."
  docker buildx build \
    --platform "$PLATFORMS" \
    -t "$service:$TAG" \
    -f "$service/Dockerfile" \
    --load \
    .
done

echo "✅ All services built!"
```

### Способ 3: Pull с GHCR

Самый простой способ - использовать уже собранные образы:

```bash
# Pull образы с GitHub Container Registry
docker pull ghcr.io/itcaat/url-shortener-demo/api-gateway:latest
docker pull ghcr.io/itcaat/url-shortener-demo/shortener-service:latest
docker pull ghcr.io/itcaat/url-shortener-demo/redirect-service:latest
docker pull ghcr.io/itcaat/url-shortener-demo/analytics-service:latest
docker pull ghcr.io/itcaat/url-shortener-demo/frontend:latest
```

## 🔧 Проверка платформы образа

```bash
# Проверить какая платформа у образа
docker image inspect api-gateway:latest | jq '.[].Architecture'

# Проверить все платформы в manifest
docker buildx imagetools inspect ghcr.io/itcaat/url-shortener-demo/api-gateway:latest
```

## 📋 Решение проблем

### Ошибка: "no matching manifest for linux/arm64/v8"

**Причина:** Образ собран только для AMD64

**Решение:**

```bash
# Вариант 1: Pull multi-platform образ с GHCR
docker-compose pull
docker-compose up

# Вариант 2: Пересобрать локально для вашей платформы
docker-compose up --build

# Вариант 3: Явно указать платформу в docker-compose.yml
services:
  api-gateway:
    platform: linux/arm64  # для Mac M1/M2/M3
    # или
    platform: linux/amd64  # для Intel
```

### Медленная сборка для другой платформы

При кросс-компиляции (AMD64 на ARM64 или наоборот) сборка будет медленнее из-за эмуляции QEMU.

**Оптимизация:**

```bash
# Собирайте только для своей платформы локально
docker-compose build

# Multi-platform оставьте для CI/CD
```

### Ошибка: "multiple platforms feature is currently not supported"

Docker Compose не поддерживает multi-platform build напрямую.

**Решение:**

```bash
# Используйте docker-compose для локальной разработки (одна платформа)
docker-compose up --build

# Используйте buildx для multi-platform
docker buildx build --platform linux/amd64,linux/arm64 ...
```

## 🎁 Best Practices

### Локальная разработка

```bash
# ✅ Используйте docker-compose (автоматически выбирает платформу)
docker-compose up --build

# ❌ Не нужно buildx для локальной разработки
```

### CI/CD (GitHub Actions)

```yaml
# ✅ Используйте multi-platform для production
platforms: linux/amd64,linux/arm64

# ✅ Кэшируйте слои
cache-from: type=gha
cache-to: type=gha,mode=max
```

### Production Deployment

```bash
# ✅ Pull образы с правильной платформой
docker pull ghcr.io/itcaat/url-shortener-demo/api-gateway:latest

# Docker автоматически выберет правильный вариант из manifest list
```

## 📊 Размеры образов

Multi-platform manifest содержит обе версии:

```
api-gateway:latest (manifest list)
├── linux/amd64 → 50MB
└── linux/arm64 → 52MB
```

При pull Docker скачивает только нужную платформу!

## 🔗 Полезные ссылки

- [Docker Buildx Documentation](https://docs.docker.com/build/building/multi-platform/)
- [Docker Compose Platform](https://docs.docker.com/compose/compose-file/05-services/#platform)
- [GitHub Actions - Docker Build](https://github.com/docker/build-push-action)

## 📝 Makefile команды

```bash
# Добавьте в Makefile:
buildx-setup:
	docker buildx create --name multiplatform --use
	docker buildx inspect multiplatform --bootstrap

buildx-build:
	docker buildx build --platform linux/amd64,linux/arm64 \
	  -f api-gateway/Dockerfile -t api-gateway:multiplatform .
```

---

**💡 Рекомендация:** Для локальной разработки используйте `docker-compose`. Для production builds используйте GitHub Actions с multi-platform support.
