# 🚀 CI/CD с GitHub Actions

Проект использует GitHub Actions для автоматической сборки Docker образов с динамической матрицой.

## 📋 Workflows

### 1. Build Pull Request (`build-pr.yml`)

**Триггеры:**
- Создание/обновление Pull Request в `main`
- Ручной запуск (`workflow_dispatch`)

**Что делает:**
1. **Определяет измененные сервисы** - анализирует какие сервисы были изменены в PR
2. **Собирает только измененные сервисы** - использует динамическую матрицу
3. **Пушит образы с тегами PR** - например, `pr-123`, `pr-123-sha123abc`
4. **Добавляет комментарий в PR** - с информацией о собранных образах

**Особенности:**
- Если изменился `pkg/` - пересобираются все Go сервисы
- Использует GitHub Container Registry (`ghcr.io`)
- Кэширует слои для ускорения сборки

**Пример тегов:**
```
ghcr.io/itcaat/url-shortener-demo/api-gateway:pr-123
ghcr.io/itcaat/url-shortener-demo/api-gateway:pr-123-abc1234
```

### 2. Build Main (`build-main.yml`)

**Триггеры:**
- Push в ветку `main` (после мержа PR)
- Ручной запуск (`workflow_dispatch`)

**Что делает:**
1. **Определяет измененные сервисы** - анализирует что изменилось в последнем коммите
2. **Собирает только измененные сервисы**
3. **Пушит образы с production тегами** - `latest`, `main-sha123abc`

**Пример тегов:**
```
ghcr.io/itcaat/url-shortener-demo/api-gateway:latest
ghcr.io/itcaat/url-shortener-demo/api-gateway:main
ghcr.io/itcaat/url-shortener-demo/api-gateway:main-abc1234
```

### 3. Build All Services (`build-all.yml`)

**Триггеры:**
- Только ручной запуск (`workflow_dispatch`)

**Что делает:**
1. **Собирает ВСЕ сервисы** - независимо от изменений
2. **Позволяет указать custom тег** - через input параметр

**Использование:**
1. Перейти в Actions → Build All Services
2. Нажать "Run workflow"
3. Указать тег (опционально, по умолчанию `manual`)

## 🔧 Как это работает

### Динамическая матрица

```yaml
jobs:
  changed-services:
    steps:
      - name: Get changed services
        uses: tj-actions/changed-files@v45
        # Автоматически определяет измененные директории
      
      - name: Set matrix
        run: |
          # Находим все сервисы (директории с Dockerfile)
          ALL_SERVICES=$(find . -maxdepth 2 -name "Dockerfile" -type f | ...)
          
          # Если изменился pkg/ - пересобираем все Go сервисы
          if echo "$CHANGED_DIRS" | grep -q "pkg"; then
            SERVICES=$(find . -maxdepth 2 -name "go.mod" -type f | ...)
          else
            # Фильтруем только измененные сервисы с Dockerfile
            SERVICES=$(echo "$CHANGED_DIRS" | jq ...)
          fi
          
          echo "matrix={\"service\":$SERVICES}" >> "$GITHUB_OUTPUT"
  
  build:
    needs: [changed-services]
    strategy:
      matrix: ${{ fromJSON(needs.changed-services.outputs.matrix) }}
    steps:
      - name: Build ${{ matrix.service }}
        # Сборка только измененных сервисов
```

**Преимущества автоматического определения:**
- ✅ **Не нужно обновлять workflow** при добавлении новых сервисов
- ✅ **Автоматически находит все сервисы** по наличию Dockerfile
- ✅ **Умная фильтрация** - только сервисы с изменениями
- ✅ **Поддержка shared libraries** - изменение pkg/ → пересборка всех Go сервисов

### Логика пересборки

**1. Если изменился `pkg/tracing/`:**
```json
{
  "service": [
    "api-gateway",
    "shortener-service", 
    "redirect-service",
    "analytics-service"
  ]
}
```

**2. Если изменился только `api-gateway/`:**
```json
{
  "service": ["api-gateway"]
}
```

**3. Если изменились `api-gateway/` и `frontend/`:**
```json
{
  "service": ["api-gateway", "frontend"]
}
```

## 📦 Структура образов в GHCR

```
ghcr.io/itcaat/url-shortener-demo/
├── api-gateway/
│   ├── :latest
│   ├── :main
│   ├── :pr-123
│   └── :main-abc1234
├── shortener-service/
│   ├── :latest
│   └── ...
├── redirect-service/
├── analytics-service/
└── frontend/
```

## 🔐 Permissions

Workflows требуют следующих permissions:

```yaml
permissions:
  contents: read        # Чтение кода
  packages: write       # Запись в GHCR
  pull-requests: write  # Комментарии в PR (только build-pr.yml)
```

## 💡 Оптимизации

### 1. Layer Caching

Используется GitHub Actions Cache для кэширования Docker слоев:

```yaml
cache-from: type=gha,scope=${{ matrix.service }}
cache-to: type=gha,mode=max,scope=${{ matrix.service }}
```

**Преимущества:**
- Ускорение сборки на 50-80%
- Меньше нагрузки на runners
- Экономия времени разработчиков

### 2. Параллельная сборка

Несколько сервисов собираются параллельно благодаря матрице:

```yaml
strategy:
  fail-fast: false
  matrix: 
    service: [api-gateway, shortener-service, ...]
```

### 3. Умное определение изменений

Игнорируем файлы, не влияющие на сборку:

```yaml
files_ignore: |
  **/*.md
  .github/**
```

## 🧪 Тестирование локально

### Проверить какие сервисы будут собраны:

```bash
# Установить tj-actions/changed-files локально
npm install -g @actions/changed-files

# Или использовать git diff
git diff --name-only main | grep -E '^(api-gateway|shortener-service|redirect-service|analytics-service|frontend|pkg)/'
```

### Собрать образы локально:

```bash
# Собрать конкретный сервис
docker build -t api-gateway:local -f api-gateway/Dockerfile .

# Собрать все сервисы
docker-compose build
```

## 📊 Примеры использования

### Сценарий 1: Изменили код api-gateway

1. Создаете PR
2. GitHub Actions определяет изменения в `api-gateway/`
3. Собирается **только** `api-gateway`
4. Образ пушится с тегом `pr-123`
5. Комментарий добавляется в PR

### Сценарий 2: Обновили pkg/tracing

1. Создаете PR с изменениями в `pkg/tracing/`
2. GitHub Actions определяет изменения в общем пакете
3. Собираются **все Go сервисы**: api-gateway, shortener-service, redirect-service, analytics-service
4. Образы пушатся с тегами `pr-123`
5. 4 комментария добавляются в PR

### Сценарий 3: Мерж в main

1. PR мержится в main
2. GitHub Actions определяет что изменилось
3. Собираются только измененные сервисы
4. Образы пушатся с тегами `latest` и `main-sha`

### Сценарий 4: Ручная сборка всех сервисов

1. Actions → Build All Services → Run workflow
2. Вводите tag: `v1.0.0`
3. Собираются **все 5 сервисов**
4. Образы пушатся с тегами `v1.0.0`, `latest`, и `sha`

## 🔍 Debugging

### Проверить логи сборки:

1. Перейти в Actions
2. Выбрать нужный workflow run
3. Кликнуть на job (например, "Build api-gateway")
4. Посмотреть логи каждого step

### Проверить какие сервисы были определены:

В step "List all changed files" видно какие файлы изменились:

```
Changed files: ["api-gateway","pkg"]
Services to build: ["api-gateway","shortener-service","redirect-service","analytics-service"]
```

### Проверить собранные образы:

```bash
# Список всех образов в GHCR
gh api /users/itcaat/packages?package_type=container

# Подробная информация об образе
gh api /users/itcaat/packages/container/url-shortener-demo%2Fapi-gateway
```

## 📝 Best Practices

### 1. Именование веток

Используйте понятные имена для веток:
```
feature/add-metrics
fix/redis-connection
refactor/tracing-package
```

### 2. Commit messages

Четкие commit messages помогают в debugging:
```
feat(api-gateway): add health check endpoint
fix(pkg/tracing): correct jaeger port
refactor(all): update dependencies
```

### 3. Pull Requests

- Создавайте небольшие PR (изменения в 1-2 сервисах)
- Проверяйте комментарии от GitHub Actions с образами
- Тестируйте образы перед мержем

### 4. Secrets

Не требуются дополнительные secrets:
- `GITHUB_TOKEN` - автоматически предоставляется
- Permissions настроены в workflow файлах

## 🚀 Деплой образов

### Pull из GHCR:

```bash
# Логин (требуется Personal Access Token с read:packages)
echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin

# Pull образа
docker pull ghcr.io/itcaat/url-shortener-demo/api-gateway:latest

# Run
docker run -p 3000:3000 ghcr.io/itcaat/url-shortener-demo/api-gateway:latest
```

### Использование в docker-compose:

```yaml
services:
  api-gateway:
    image: ghcr.io/itcaat/url-shortener-demo/api-gateway:latest
    ports:
      - "3000:3000"
```

## 📚 Ссылки

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [GitHub Container Registry](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
- [Docker Build Push Action](https://github.com/docker/build-push-action)
- [Changed Files Action](https://github.com/tj-actions/changed-files)

---

**✨ CI/CD настроен и готов к использованию!**
