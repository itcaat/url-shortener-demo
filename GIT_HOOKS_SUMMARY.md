# 🪝 Git Hooks - Quick Summary

## 📝 Что это?

Автоматическое добавление ID GitHub Issue в commit message на основе названия ветки.

## 🚀 Установка

```bash
make hook
```

## 🎯 Поддерживаемые форматы

| Формат ветки | Результат в commit |
|--------------|-------------------|
| `feature/123-description` | `[#123] your message` |
| `fix/#456-bug-fix` | `[#456] your message` |
| `feat/GH-789-improvement` | `[#789] your message` |
| `1-demo-build-matrix` ⭐ | `[#1] your message` |
| `42-quick-fix` ⭐ | `[#42] your message` |
| `bugfix/TASK-999-legacy` | `[TASK-999] your message` |

⭐ **Краткий формат** - ветки без префикса, только `число-описание`

## 💡 Пример использования

```bash
# 1. Установить hook
make hook

# 2. Создать ветку с ID issue
git checkout -b feature/42-add-auth

# 3. Обычный коммит (БЕЗ ID)
git commit -m "add JWT authentication"

# 4. Результат (С ID)
✅ [#42] add JWT authentication
```

## 🔗 Интеграция с GitHub

### Автоматические ссылки
```bash
git commit -m "fix bug"
# → [#123] fix bug
# В GitHub: ссылка на issue #123
```

### Автоматическое закрытие issues
```bash
git commit -m "fixes login bug"
# → [#123] fixes login bug
# При мерже: issue #123 закроется
```

**Ключевые слова для закрытия:**
- `closes #123`
- `fixes #123`
- `resolves #123`

## 🎨 Workflow примеры

### Новая фича
```bash
git checkout -b feature/42-user-profiles
git commit -m "add user profile page"
# → [#42] add user profile page
```

### Исправление бага
```bash
git checkout -b fix/99-memory-leak
git commit -m "closes memory leak in analytics"
# → [#99] closes memory leak in analytics
# При мерже → issue #99 автоматически закроется
```

### Улучшение
```bash
git checkout -b feat/123-performance
git commit -m "optimize database queries"
# → [#123] optimize database queries
```

### Краткий формат (без префикса)
```bash
git checkout -b 1-demo-build-matrix
git commit -m "add dynamic matrix to CI"
# → [#1] add dynamic matrix to CI

git checkout -b 42-hotfix
git commit -m "fixes critical security issue"
# → [#42] fixes critical security issue
# При мерже → issue #42 закроется
```

## ⚙️ Как это работает

1. **Hook срабатывает** при каждом `git commit`
2. **Извлекает ID** из названия ветки через regex
3. **Проверяет** наличие ID в сообщении
4. **Добавляет ID** в начало, если его нет

## 🚫 Когда hook НЕ работает

- ❌ Ветка `main`, `master`, `develop`
- ❌ Merge commits
- ❌ ID уже есть в сообщении
- ❌ Неподдерживаемый формат названия ветки

## 🛠️ Управление

### Установить
```bash
make hook
```

### Временно отключить
```bash
git commit --no-verify -m "message"
```

### Удалить
```bash
rm .git/hooks/prepare-commit-msg
```

### Обновить
```bash
make hook  # После изменения .git-hooks/prepare-commit-msg
```

## 🐛 Troubleshooting

### Hook не работает
```bash
# Проверить установлен ли
ls -la .git/hooks/prepare-commit-msg

# Переустановить
make hook
```

### ID не извлекается
```bash
# Проверить формат ветки
git branch --show-current

# Поддерживаемые форматы:
# feature/123-description  ✅
# fix/#456-bug            ✅
# feat/GH-789-feature     ✅
# 1-demo-build-matrix     ✅ (краткий формат)
# 42-quick-fix            ✅ (краткий формат)
# bugfix/TASK-123-fix     ✅ (legacy)
# 
# НЕ поддерживаемые:
# feature-123             ❌ (нет /)
# my-feature              ❌ (нет цифр)
# update-readme           ❌ (нет цифр)
```

## 📚 Документация

- **Полная документация:** `.git-hooks/README.md`
- **Git hook файл:** `.git-hooks/prepare-commit-msg`
- **Команды:** `make help` → смотреть команду `hook`

## 🎁 Преимущества

✅ **Автоматизация** - не нужно вручную добавлять ID  
✅ **Консистентность** - единый формат коммитов  
✅ **Traceability** - легко найти issue по коммиту  
✅ **GitHub Integration** - автоматические ссылки  
✅ **Auto-close** - автоматическое закрытие issues  

## 💼 Best Practices

1. **Всегда создавайте ветки с ID issue**
   ```bash
   ✅ git checkout -b feature/123-new-feature
   ❌ git checkout -b new-feature
   ```

2. **Используйте осмысленные commit messages**
   ```bash
   ✅ git commit -m "add user authentication with JWT"
   ❌ git commit -m "update"
   ```

3. **Один issue - одна ветка**
   ```bash
   ✅ feature/123-auth
   ❌ feature/123-124-mixed
   ```

4. **Используйте ключевые слова для закрытия**
   ```bash
   git commit -m "fixes login bug"
   # При мерже issue закроется автоматически
   ```

## 🔄 Совместимость

- ✅ macOS (Bash/Zsh)
- ✅ Linux (Bash/Zsh)
- ✅ Windows (Git Bash)
- ✅ GitHub Issues
- ✅ Legacy support (Jira, etc.)

## 📊 Примеры в проекте

После установки hook все ваши коммиты в ветках с ID будут автоматически форматироваться:

```bash
# В ветке feature/42-add-tracing
git commit -m "add OpenTelemetry instrumentation"
# → [#42] add OpenTelemetry instrumentation

# В ветке fix/99-kafka-bug
git commit -m "fixes kafka consumer group rebalance"
# → [#99] fixes kafka consumer group rebalance
# При мерже → issue #99 закроется

# В ветке feat/123-ci-cd
git commit -m "add GitHub Actions workflows"
# → [#123] add GitHub Actions workflows
```

---

**🎉 Готово к использованию! Установите: `make hook`**

**📖 Полная документация:** `.git-hooks/README.md`
