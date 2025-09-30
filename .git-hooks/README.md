# Git Hooks

Этот проект использует git hooks для автоматизации workflow.

## 📋 Доступные hooks

### `prepare-commit-msg`

Автоматически добавляет ID таска в commit message на основе названия ветки.

**Поддерживаемые форматы веток:**
```
# С префиксом (рекомендуется)
feature/123-description      → [#123] your commit message
bugfix/456-fix-bug          → [#456] your commit message
feat/#789-new-feature       → [#789] your commit message
fix/GH-999-hotfix           → [#999] your commit message

# Без префикса (краткий формат)
1-demo-build-matrix         → [#1] your commit message
42-quick-fix                → [#42] your commit message

# Legacy support (Jira, etc.)
feature/TASK-123-description → [TASK-123] your commit message
bugfix/PROJ-456-fix         → [PROJ-456] your commit message
```

**Игнорируемые ветки:**
- `main`
- `master`
- `develop`
- `dev`

## 🔧 Установка

```bash
# Установить все git hooks
make hook

# Или вручную
chmod +x .git-hooks/prepare-commit-msg
cp .git-hooks/prepare-commit-msg .git/hooks/prepare-commit-msg
```

## 💡 Примеры использования

### Пример 1: Стандартный workflow (GitHub Issues)

```bash
# Создать ветку с номером issue
git checkout -b feature/123-add-metrics

# Сделать изменения
vim api-gateway/main.go

# Коммит БЕЗ указания ID
git commit -m "add metrics endpoint"

# Результат: [#123] add metrics endpoint
```

### Пример 2: ID уже есть в сообщении

```bash
git checkout -b bugfix/456-fix-redis

# Коммит с ID в сообщении
git commit -m "#456: fix redis connection"

# Результат: #456: fix redis connection (не изменится)
```

### Пример 3: Разные форматы веток

```bash
# Стандартный GitHub issue
git checkout -b feature/123-new-ui
git commit -m "update dashboard"
# → [#123] update dashboard

# С префиксом #
git checkout -b fix/#456-memory-leak
git commit -m "fix memory leak"
# → [#456] fix memory leak

# С префиксом GH
git checkout -b feat/GH-789-integration
git commit -m "integrate payment gateway"
# → [#789] integrate payment gateway

# Legacy Jira поддержка
git checkout -b feature/PROJ-999-auth
git commit -m "add authentication"
# → [PROJ-999] add authentication
```

## 🔍 Как это работает

1. **Hook срабатывает** перед созданием commit message
2. **Получает название ветки**: `git symbolic-ref --short HEAD`
3. **Извлекает ID** (4 паттерна):
   - С префиксом: `/+#?([0-9]+)-` → `feature/123-fix` → `#123`
   - GH-prefix: `/GH-([0-9]+)` → `fix/GH-456-bug` → `#456`
   - Без префикса: `^([0-9]+)-` → `1-demo-build` → `#1`
   - Legacy: `([A-Z]+-[0-9]+)` → `TASK-123-feat` → `TASK-123`
4. **Проверяет commit message**: есть ли уже ID
5. **Добавляет ID** в начало сообщения: `[#123] original message`

## ⚙️ Конфигурация

### Изменить формат ID в сообщении

Отредактируйте `.git-hooks/prepare-commit-msg`:

```bash
# Текущий формат: [TASK-123] message
echo "[$TASK_ID] $COMMIT_MSG" > "$COMMIT_MSG_FILE"

# Альтернативный формат: TASK-123: message
echo "$TASK_ID: $COMMIT_MSG" > "$COMMIT_MSG_FILE"

# Формат в конце: message (TASK-123)
echo "$COMMIT_MSG ($TASK_ID)" > "$COMMIT_MSG_FILE"
```

### Изменить regex для извлечения ID

Текущие regex:
- GitHub Issues: `/?#?([0-9]+)-` - находит `123` → `#123`
- GH-prefix: `/GH-([0-9]+)` - находит `GH-123` → `#123`
- Legacy: `([A-Z]+-[0-9]+)` - находит `ABC-123` → `ABC-123`

```bash
# Только цифры без #
if [[ "$BRANCH_NAME" =~ /([0-9]+)- ]]; then
  TASK_ID="${BASH_REMATCH[1]}"
fi

# Формат task_123
if [[ "$BRANCH_NAME" =~ task_([0-9]+) ]]; then
  TASK_ID="task_${BASH_REMATCH[1]}"
fi

# Формат ISSUE-123 (без # в результате)
if [[ "$BRANCH_NAME" =~ ISSUE-([0-9]+) ]]; then
  TASK_ID="ISSUE-${BASH_REMATCH[1]}"
fi
```

## 🚫 Отключение hook

### Временно для одного коммита

```bash
git commit --no-verify -m "your message"
```

### Полностью удалить

```bash
rm .git/hooks/prepare-commit-msg
```

## 🔄 Обновление hooks

После обновления файлов в `.git-hooks/`:

```bash
make hook
```

Это скопирует обновленные hooks в `.git/hooks/`.

## 📝 Best Practices

### 1. Всегда создавайте ветки с ID issue

```bash
# ✅ Хорошо - GitHub issue
git checkout -b feature/123-new-feature
git checkout -b fix/456-bug-fix
git checkout -b feat/#789-improvement

# ❌ Плохо
git checkout -b new-feature
```

### 2. Используйте осмысленные commit messages

Hook добавит ID, вы фокусируйтесь на описании:

```bash
# ✅ Хорошо
git commit -m "add user authentication with JWT"
# → [#123] add user authentication with JWT

# ❌ Плохо
git commit -m "update"
# → [#123] update
```

### 3. Один issue - одна ветка

```bash
# ✅ Хорошо
feature/123-auth
feature/124-metrics

# ❌ Плохо (несколько ID в названии)
feature/123-124-mixed
```

## 🐛 Troubleshooting

### Hook не работает

```bash
# Проверить установлен ли hook
ls -la .git/hooks/prepare-commit-msg

# Проверить права
chmod +x .git/hooks/prepare-commit-msg

# Переустановить
make hook
```

### ID не извлекается

```bash
# Проверить название ветки
git branch --show-current

# Поддерживаемые форматы:
# ✅ feature/123-description  → #123
# ✅ fix/#456-bug            → #456
# ✅ feat/GH-789-feature     → #789
# ✅ 1-demo-build-matrix     → #1
# ✅ 42-quick-fix            → #42
# ✅ bugfix/TASK-123-fix     → TASK-123

# НЕ поддерживаемые:
# ❌ my-feature-branch       → нет цифр
# ❌ update-readme           → нет цифр
# ❌ feature-123             → нет дефиса после цифр

# Если формат другой, обновите regex в hook
```

### Hook добавляет ID дважды

Hook проверяет наличие ID в сообщении. Если это происходит:

1. Проверьте regex в условии проверки
2. Убедитесь что формат ID совпадает

## 🔗 Интеграция с GitHub

Commits с ID issue автоматически:
- ✅ **Линкуются с GitHub Issues** - `[#123]` создает ссылку на issue #123
- ✅ **Закрывают issues** - используйте `closes #123`, `fixes #123`, `resolves #123`
- ✅ **Отображаются в PR** - все связанные issues показываются в Pull Request
- ✅ **Упрощают code review** - сразу видно контекст изменений

**Пример автоматического закрытия:**
```bash
# Эта ветка и коммит закроют issue #123 при мерже в main
git checkout -b fix/123-bug-fix
git commit -m "closes connection leak"
# Результат: [#123] closes connection leak
# При мерже → issue #123 автоматически закроется
```

## 📚 Дополнительные ресурсы

- [Git Hooks Documentation](https://git-scm.com/docs/githooks)
- [Customizing Git - Git Hooks](https://git-scm.com/book/en/v2/Customizing-Git-Git-Hooks)
- [prepare-commit-msg Hook](https://git-scm.com/docs/githooks#_prepare_commit_msg)

---

**💡 Tip:** После установки hook'а попробуйте создать тестовую ветку и коммит для проверки работы!
