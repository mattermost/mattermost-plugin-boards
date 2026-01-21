# Быстрый старт: Создание релиза

## 🚀 Автоматический релиз (GitHub Actions)

### Шаг 1: Обновите версию
```bash
# Откройте plugin.json и измените версию
nano plugin.json
# Измените: "version": "9.2.3"
```

### Шаг 2: Закоммитьте и запушьте
```bash
git add plugin.json
git commit -m "Release v9.2.3"
git push origin main:release
```

### Шаг 3: Дождитесь сборки
- Перейдите в GitHub → Actions
- Дождитесь завершения workflow "Release Build"
- Релиз появится в разделе Releases
- **Создается bundle для Linux AMD64** (~46 MB)

### Шаг 4: Скачайте и установите
```bash
# На вашем сервере
cd /tmp
wget https://github.com/fambear/mattermost-plugin-boards/releases/download/v9.2.3/boards-9.2.3.tar.gz

# Установите плагин
cd /opt/mattermost/plugins
rm -rf boards  # Удалите старую версию
tar -xzf /tmp/boards-9.2.3.tar.gz

# Перезапустите Mattermost
systemctl restart mattermost
```

---

## 🔧 Локальная сборка

### Linux/macOS
```bash
./scripts/build-release.sh
```

### Windows
```powershell
.\scripts\build-release.ps1
```

### Результат
Файл будет создан в: `dist/boards-{version}.tar.gz`

---

## 📋 Checklist перед релизом

- [ ] Обновлена версия в `plugin.json`
- [ ] Все изменения закоммичены
- [ ] Тесты проходят: `make ci`
- [ ] Локальная сборка работает: `make dist-linux`
- [ ] Changelog обновлен (если есть)

---

## 🔍 Проверка релиза

После установки на сервере:

1. Откройте Mattermost
2. System Console → Plugins → Plugin Management
3. Найдите "Mattermost Boards"
4. Проверьте версию плагина
5. Убедитесь что плагин активен

---

## ❓ Troubleshooting

### Релиз не создается
```bash
# Проверьте логи GitHub Actions
# Убедитесь что версия корректна
jq -r '.version' plugin.json
```

### Сборка падает локально
```bash
# Проверьте зависимости
go version
npm --version

# Переустановите зависимости
cd webapp && npm ci
```

### Плагин не работает после установки
```bash
# Проверьте логи Mattermost
tail -f /opt/mattermost/logs/mattermost.log

# Проверьте права на файлы
chown -R mattermost:mattermost /opt/mattermost/plugins/boards
```

---

## 📚 Дополнительная информация

- Полная инструкция: [RELEASE.md](RELEASE.md)
- Документация по сборке: [README.md](README.md)
- GitHub Actions workflow: [.github/workflows/release.yml](.github/workflows/release.yml)

