#!/bin/bash
set -e

# Скрипт для локальной сборки релиза
# Использование: ./scripts/build-release.sh

echo "🔨 Building Mattermost Boards Plugin Release..."
echo ""

# Проверка наличия необходимых инструментов
command -v jq >/dev/null 2>&1 || { echo "❌ jq is required but not installed. Install it with: sudo apt-get install jq"; exit 1; }
command -v go >/dev/null 2>&1 || { echo "❌ go is required but not installed."; exit 1; }
command -v npm >/dev/null 2>&1 || { echo "❌ npm is required but not installed."; exit 1; }

# Извлечение версии из plugin.json
VERSION=$(jq -r '.version' plugin.json)
echo "📦 Version: $VERSION"
echo ""

# Проверка что мы в корне проекта
if [ ! -f "plugin.json" ]; then
    echo "❌ Error: plugin.json not found. Run this script from the project root."
    exit 1
fi

# Установка зависимостей
echo "📥 Installing dependencies..."
cd webapp
npm ci
cd ..
echo "✅ Dependencies installed"
echo ""

# Сборка плагина
echo "🔧 Building plugin for Linux AMD64..."
make dist-linux
echo "✅ Build complete"
echo ""

# Проверка результата
BUNDLE_NAME="boards-${VERSION}.tar.gz"
if [ -f "dist/${BUNDLE_NAME}" ]; then
    echo "✅ Release bundle created successfully!"
    echo ""
    echo "📦 Bundle: dist/${BUNDLE_NAME}"
    echo "📊 Size: $(du -h dist/${BUNDLE_NAME} | cut -f1)"
    echo ""
    echo "🚀 You can now upload this file to your Mattermost server"
    echo ""
    echo "To install on server:"
    echo "  scp dist/${BUNDLE_NAME} user@server:/tmp/"
    echo "  ssh user@server"
    echo "  cd /opt/mattermost/plugins"
    echo "  tar -xzf /tmp/${BUNDLE_NAME}"
    echo "  systemctl restart mattermost"
else
    echo "❌ Error: Bundle not found at dist/${BUNDLE_NAME}"
    exit 1
fi

