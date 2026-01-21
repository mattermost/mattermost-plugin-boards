# Инструкция по созданию релиза

## Автоматическая сборка релиза

Проект настроен на автоматическую сборку и публикацию релизов при пуше в ветку `release`.

### Процесс создания релиза

1. **Обновите версию в `plugin.json`**
   ```bash
   # Отредактируйте файл plugin.json и измените поле "version"
   # Например: "version": "9.2.3"
   ```

2. **Закоммитьте изменения**
   ```bash
   git add plugin.json
   git commit -m "Bump version to 9.2.3"
   ```

3. **Запушьте в ветку release**
   ```bash
   git push origin main:release
   # или если вы уже в ветке release:
   git push origin release
   ```

4. **Автоматическая сборка**
   
   GitHub Actions автоматически:
   - Извлечет версию из `plugin.json`
   - Соберет плагин для Linux AMD64
   - Создаст git tag `v{version}` (например, `v9.2.3`)
   - Создаст GitHub Release
   - Загрузит файл `boards-{version}.tar.gz` в релиз

### Что собирается

- **Платформа:** Linux AMD64 only
- **Файл:** `boards-{version}.tar.gz`
- **Содержимое:** Готовый плагин для установки в Mattermost

### Установка на сервер

После создания релиза, скачайте и установите плагин:

```bash
# Скачайте релиз
wget https://github.com/fambear/mattermost-plugin-boards/releases/download/v9.2.3/boards-9.2.3.tar.gz

# Распакуйте в директорию плагинов Mattermost
cd /opt/mattermost/plugins
tar -xzf boards-9.2.3.tar.gz

# Перезапустите Mattermost
systemctl restart mattermost
```

Или используйте веб-интерфейс Mattermost:
1. System Console → Plugins → Plugin Management
2. Upload Plugin
3. Выберите скачанный файл `boards-{version}.tar.gz`
4. Enable плагин

### Обновление существующего релиза

Если релиз с текущей версией уже существует, workflow:
- Не создаст новый тег
- Обновит существующий релиз
- Заменит файл артефакта на новый

### Ручной запуск

Вы также можете запустить сборку вручную:
1. Перейдите в GitHub → Actions → Release Build
2. Нажмите "Run workflow"
3. Выберите ветку `release`
4. Нажмите "Run workflow"

### Проверка статуса сборки

Статус сборки можно посмотреть:
- GitHub → Actions → Release Build
- В Summary будет информация о созданном релизе

### Требования

Для работы workflow требуется:
- ✅ Версия в `plugin.json` должна быть корректной (формат: `X.Y.Z`)
- ✅ GitHub Actions должны иметь права на создание релизов (permissions: contents: write)
- ✅ Все зависимости должны быть доступны (npm packages, go modules)

### Troubleshooting

**Проблема:** Релиз не создается
- Проверьте логи GitHub Actions
- Убедитесь что версия в `plugin.json` корректна
- Проверьте что у вас есть права на push в ветку `release`

**Проблема:** Сборка падает
- Проверьте что все зависимости установлены
- Убедитесь что код компилируется локально: `make dist-linux`

**Проблема:** Тег уже существует
- Workflow обновит существующий релиз вместо создания нового
- Если нужно пересоздать тег, удалите его вручную:
  ```bash
  git tag -d v9.2.3
  git push origin :refs/tags/v9.2.3
  ```

