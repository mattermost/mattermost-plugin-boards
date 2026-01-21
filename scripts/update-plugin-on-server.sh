#!/bin/bash
set -e

# Скрипт для автоматического обновления плагина на сервере Mattermost
# Использование: ./scripts/update-plugin-on-server.sh [version]
# Пример: ./scripts/update-plugin-on-server.sh 9.2.3

GITHUB_REPO="fambear/mattermost-plugin-boards"
MATTERMOST_PATH="/opt/mattermost"
PLUGIN_NAME="boards"

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔄 Mattermost Boards Plugin Updater${NC}"
echo ""

# Проверка прав root
if [ "$EUID" -ne 0 ]; then 
    echo -e "${RED}❌ Please run as root or with sudo${NC}"
    exit 1
fi

# Получение версии
if [ -z "$1" ]; then
    echo -e "${YELLOW}📥 Fetching latest release version...${NC}"
    VERSION=$(curl -s "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"v([^"]+)".*/\1/')
    if [ -z "$VERSION" ]; then
        echo -e "${RED}❌ Failed to fetch latest version${NC}"
        exit 1
    fi
else
    VERSION=$1
fi

echo -e "${GREEN}📦 Version to install: ${VERSION}${NC}"
echo ""

# Формирование URL для скачивания
DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/download/v${VERSION}/${PLUGIN_NAME}-${VERSION}.tar.gz"
TEMP_FILE="/tmp/${PLUGIN_NAME}-${VERSION}.tar.gz"

# Скачивание релиза
echo -e "${YELLOW}📥 Downloading plugin from GitHub...${NC}"
echo "URL: ${DOWNLOAD_URL}"
if ! curl -L -f -o "${TEMP_FILE}" "${DOWNLOAD_URL}"; then
    echo -e "${RED}❌ Failed to download plugin${NC}"
    echo "Please check if release v${VERSION} exists at:"
    echo "https://github.com/${GITHUB_REPO}/releases"
    exit 1
fi
echo -e "${GREEN}✅ Downloaded successfully${NC}"
echo ""

# Создание бэкапа текущей версии
if [ -d "${MATTERMOST_PATH}/plugins/${PLUGIN_NAME}" ]; then
    BACKUP_DIR="${MATTERMOST_PATH}/plugins/${PLUGIN_NAME}.backup.$(date +%Y%m%d_%H%M%S)"
    echo -e "${YELLOW}💾 Creating backup...${NC}"
    cp -r "${MATTERMOST_PATH}/plugins/${PLUGIN_NAME}" "${BACKUP_DIR}"
    echo -e "${GREEN}✅ Backup created: ${BACKUP_DIR}${NC}"
    echo ""
fi

# Остановка Mattermost
echo -e "${YELLOW}⏸️  Stopping Mattermost...${NC}"
systemctl stop mattermost
echo -e "${GREEN}✅ Mattermost stopped${NC}"
echo ""

# Удаление старой версии
if [ -d "${MATTERMOST_PATH}/plugins/${PLUGIN_NAME}" ]; then
    echo -e "${YELLOW}🗑️  Removing old plugin version...${NC}"
    rm -rf "${MATTERMOST_PATH}/plugins/${PLUGIN_NAME}"
    echo -e "${GREEN}✅ Old version removed${NC}"
    echo ""
fi

# Установка новой версии
echo -e "${YELLOW}📦 Installing new plugin version...${NC}"
cd "${MATTERMOST_PATH}/plugins"
tar -xzf "${TEMP_FILE}"
echo -e "${GREEN}✅ Plugin extracted${NC}"
echo ""

# Установка правильных прав
echo -e "${YELLOW}🔐 Setting permissions...${NC}"
chown -R mattermost:mattermost "${MATTERMOST_PATH}/plugins/${PLUGIN_NAME}"
echo -e "${GREEN}✅ Permissions set${NC}"
echo ""

# Запуск Mattermost
echo -e "${YELLOW}▶️  Starting Mattermost...${NC}"
systemctl start mattermost
echo -e "${GREEN}✅ Mattermost started${NC}"
echo ""

# Ожидание запуска
echo -e "${YELLOW}⏳ Waiting for Mattermost to start...${NC}"
sleep 5

# Проверка статуса
if systemctl is-active --quiet mattermost; then
    echo -e "${GREEN}✅ Mattermost is running${NC}"
else
    echo -e "${RED}❌ Mattermost failed to start${NC}"
    echo "Check logs: journalctl -u mattermost -n 50"
    exit 1
fi

# Очистка
echo -e "${YELLOW}🧹 Cleaning up...${NC}"
rm -f "${TEMP_FILE}"
echo -e "${GREEN}✅ Cleanup complete${NC}"
echo ""

echo -e "${GREEN}🎉 Plugin updated successfully!${NC}"
echo ""
echo -e "${BLUE}Plugin version: ${VERSION}${NC}"
echo -e "${BLUE}Installation path: ${MATTERMOST_PATH}/plugins/${PLUGIN_NAME}${NC}"
if [ -n "${BACKUP_DIR}" ]; then
    echo -e "${BLUE}Backup location: ${BACKUP_DIR}${NC}"
fi
echo ""
echo -e "${YELLOW}📝 Next steps:${NC}"
echo "1. Open Mattermost in your browser"
echo "2. Go to System Console → Plugins → Plugin Management"
echo "3. Verify that 'Mattermost Boards' shows version ${VERSION}"
echo "4. Ensure the plugin is enabled"
echo ""
echo -e "${YELLOW}💡 To rollback to previous version:${NC}"
if [ -n "${BACKUP_DIR}" ]; then
    echo "systemctl stop mattermost"
    echo "rm -rf ${MATTERMOST_PATH}/plugins/${PLUGIN_NAME}"
    echo "mv ${BACKUP_DIR} ${MATTERMOST_PATH}/plugins/${PLUGIN_NAME}"
    echo "systemctl start mattermost"
fi

