# ⚡ CI/CD Optimization Guide

Опции для ускорения multi-platform сборки в GitHub Actions.

## 🎯 Текущая ситуация

**Текущая конфигурация:**
```yaml
runs-on: ubuntu-latest  # AMD64 runner
platforms: linux/amd64,linux/arm64
```

**Время сборки:**
- AMD64: ~2-3 минуты (нативная)
- ARM64: ~5-8 минут (QEMU эмуляция)
- **Итого:** ~8-10 минут для обеих платформ

**⚠️ Важно:** Медленнее только **СБОРКА** в CI/CD, не работа образа!

## 🚀 Опции ускорения

### Опция 1: Параллельная сборка на разных раннерах (рекомендуется)

Собираем AMD64 и ARM64 на отдельных раннерах параллельно, затем объединяем.

**Преимущества:**
- ✅ Каждая платформа собирается нативно
- ✅ В 2 раза быстрее (параллельно)
- ✅ Бесплатно (GitHub hosted runners)

**Пример workflow:**

```yaml
jobs:
  build-amd64:
    runs-on: ubuntu-latest
    steps:
      - name: Build AMD64
        uses: docker/build-push-action@v6
        with:
          platforms: linux/amd64
          outputs: type=image,push=true,name-canonical=true,push-by-digest=true
          
  build-arm64:
    runs-on: ubuntu-latest  # Использует QEMU
    # ИЛИ для скорости:
    # runs-on: [self-hosted, ARM64]  # Если есть свой ARM раннер
    steps:
      - name: Build ARM64
        uses: docker/build-push-action@v6
        with:
          platforms: linux/arm64
          outputs: type=image,push=true,name-canonical=true,push-by-digest=true
  
  merge:
    needs: [build-amd64, build-arm64]
    runs-on: ubuntu-latest
    steps:
      - name: Create manifest
        run: |
          docker buildx imagetools create -t $IMAGE:latest \
            $IMAGE@$DIGEST_AMD64 \
            $IMAGE@$DIGEST_ARM64
```

### Опция 2: GitHub ARM64 Runners (платно)

GitHub предлагает ARM64 hosted runners (beta).

**Конфигурация:**
```yaml
strategy:
  matrix:
    include:
      - platform: linux/amd64
        runner: ubuntu-latest
      - platform: linux/arm64
        runner: ubuntu-latest-arm  # GitHub ARM64 runner
        
runs-on: ${{ matrix.runner }}
```

**Стоимость:**
- AMD64: бесплатно (2000 мин/месяц для публичных репо)
- ARM64: платно (~$0.08/минута)

### Опция 3: Self-Hosted ARM64 Runner

Свой раннер на ARM64 машине (AWS Graviton, Mac M1, Raspberry Pi).

**Настройка:**
```yaml
jobs:
  build:
    strategy:
      matrix:
        include:
          - platform: linux/amd64
            runner: ubuntu-latest
          - platform: linux/arm64
            runner: [self-hosted, ARM64]
            
    runs-on: ${{ matrix.runner }}
```

**Преимущества:**
- ✅ Нативная сборка ARM64
- ✅ Полный контроль
- ✅ Можно использовать Mac M1/M2

### Опция 4: Собирать только одну платформу по условию

Собираем ARM64 только при необходимости.

```yaml
- name: Determine platforms
  id: platforms
  run: |
    # По умолчанию только AMD64
    PLATFORMS="linux/amd64"
    
    # ARM64 только для main или при метке
    if [[ "${{ github.ref }}" == "refs/heads/main" ]] || \
       [[ "${{ contains(github.event.pull_request.labels.*.name, 'multiplatform') }}" == "true" ]]; then
      PLATFORMS="linux/amd64,linux/arm64"
    fi
    
    echo "platforms=$PLATFORMS" >> $GITHUB_OUTPUT

- name: Build
  uses: docker/build-push-action@v6
  with:
    platforms: ${{ steps.platforms.outputs.platforms }}
```

### Опция 5: Кэширование (уже используется)

Наши workflows уже используют GitHub Actions Cache:

```yaml
cache-from: type=gha,scope=${{ matrix.service }}
cache-to: type=gha,mode=max,scope=${{ matrix.service }}
```

**Дополнительная оптимизация:**

```yaml
# Используем Registry cache для еще большей скорости
cache-from: |
  type=gha,scope=${{ matrix.service }}
  type=registry,ref=ghcr.io/${{ github.repository }}/${{ matrix.service }}:buildcache
cache-to: |
  type=gha,mode=max,scope=${{ matrix.service }}
  type=registry,ref=ghcr.io/${{ github.repository }}/${{ matrix.service }}:buildcache,mode=max
```

## 📊 Сравнение опций

| Опция | Скорость | Стоимость | Сложность |
|-------|----------|-----------|-----------|
| Текущая (QEMU) | 8-10 мин | Бесплатно | Низкая ✅ |
| Параллельные раннеры | 4-5 мин | Бесплатно | Средняя |
| GitHub ARM64 | 3-4 мин | ~$5-10/месяц | Низкая |
| Self-hosted | 3-4 мин | Инфра | Высокая |
| Условная сборка | 2-3 мин* | Бесплатно | Низкая |

*только AMD64 для PR, ARM64 для main

## 💡 Рекомендации

### Для демо/личных проектов:
✅ **Текущая конфигурация** (QEMU)
- Простая
- Бесплатная
- Достаточно быстрая

### Для production проектов:
✅ **Параллельные раннеры** + **Условная сборка**
- PR: только AMD64 (быстро)
- Main: AMD64 + ARM64 (полная сборка)
- Хороший баланс скорости и стоимости

### Для больших команд:
✅ **GitHub ARM64 runners** или **Self-hosted**
- Максимальная скорость
- Стоит инвестиций

## 🔧 Пример: Параллельная сборка

<details>
<summary>Полный пример workflow</summary>

```yaml
name: Build Multi-Platform (Optimized)

on: [push, pull_request]

jobs:
  prepare:
    runs-on: ubuntu-latest
    outputs:
      services: ${{ steps.services.outputs.matrix }}
    steps:
      - uses: actions/checkout@v4
      - id: services
        run: |
          SERVICES=$(find . -maxdepth 2 -name "Dockerfile" | ...)
          echo "matrix={\"service\":$SERVICES}" >> $GITHUB_OUTPUT

  build-amd64:
    needs: prepare
    runs-on: ubuntu-latest
    strategy:
      matrix: ${{ fromJSON(needs.prepare.outputs.services) }}
    steps:
      - uses: actions/checkout@v4
      - uses: docker/setup-buildx-action@v3
      - uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      
      - name: Build AMD64
        id: build
        uses: docker/build-push-action@v6
        with:
          context: .
          file: ${{ matrix.service }}/Dockerfile
          platforms: linux/amd64
          outputs: type=image,name=ghcr.io/${{ github.repository }}/${{ matrix.service }},push-by-digest=true,name-canonical=true,push=true
      
      - name: Export digest
        run: |
          mkdir -p /tmp/digests
          digest="${{ steps.build.outputs.digest }}"
          echo "$digest" > "/tmp/digests/${digest#sha256:}"
      
      - uses: actions/upload-artifact@v4
        with:
          name: digests-amd64-${{ matrix.service }}
          path: /tmp/digests/*

  build-arm64:
    needs: prepare
    runs-on: ubuntu-latest
    strategy:
      matrix: ${{ fromJSON(needs.prepare.outputs.services) }}
    steps:
      # То же самое, но platforms: linux/arm64
      # ...

  merge:
    needs: [build-amd64, build-arm64]
    runs-on: ubuntu-latest
    strategy:
      matrix: ${{ fromJSON(needs.prepare.outputs.services) }}
    steps:
      - uses: actions/download-artifact@v4
        with:
          pattern: digests-*-${{ matrix.service }}
          merge-multiple: true
          path: /tmp/digests
      
      - uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      
      - name: Create manifest
        working-directory: /tmp/digests
        run: |
          docker buildx imagetools create \
            -t ghcr.io/${{ github.repository }}/${{ matrix.service }}:latest \
            $(printf 'ghcr.io/${{ github.repository }}/${{ matrix.service }}@sha256:%s ' *)
```

</details>

## 🎯 Итоговая рекомендация

**Для вашего проекта:**

```yaml
# Оставить текущую конфигурацию для простоты
# + Добавить условную сборку для PR

- name: Set platforms
  id: platforms
  run: |
    if [[ "${{ github.event_name }}" == "pull_request" ]]; then
      echo "platforms=linux/amd64" >> $GITHUB_OUTPUT
    else
      echo "platforms=linux/amd64,linux/arm64" >> $GITHUB_OUTPUT
    fi

- name: Build
  with:
    platforms: ${{ steps.platforms.outputs.platforms }}
```

**Результат:**
- PR: 2-3 минуты (только AMD64)
- Main: 8-10 минут (обе платформы)
- Бесплатно
- Просто

---

**💡 Вывод:** Текущая конфигурация оптимальна для большинства случаев. Медленная сборка ARM64 в CI - это нормально и один раз. Итоговый образ работает быстро!
