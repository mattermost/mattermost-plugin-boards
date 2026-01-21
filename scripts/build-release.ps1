# Скрипт для локальной сборки релиза на Windows
# Использование: .\scripts\build-release.ps1

$ErrorActionPreference = "Stop"

Write-Host "🔨 Building Mattermost Boards Plugin Release..." -ForegroundColor Cyan
Write-Host ""

# Проверка наличия необходимых инструментов
function Test-Command {
    param($Command)
    try {
        Get-Command $Command -ErrorAction Stop | Out-Null
        return $true
    } catch {
        return $false
    }
}

if (-not (Test-Command "go")) {
    Write-Host "❌ go is required but not installed." -ForegroundColor Red
    exit 1
}

if (-not (Test-Command "npm")) {
    Write-Host "❌ npm is required but not installed." -ForegroundColor Red
    exit 1
}

# Проверка что мы в корне проекта
if (-not (Test-Path "plugin.json")) {
    Write-Host "❌ Error: plugin.json not found. Run this script from the project root." -ForegroundColor Red
    exit 1
}

# Извлечение версии из plugin.json
$pluginJson = Get-Content "plugin.json" -Raw | ConvertFrom-Json
$VERSION = $pluginJson.version
Write-Host "📦 Version: $VERSION" -ForegroundColor Green
Write-Host ""

# Установка зависимостей
Write-Host "📥 Installing dependencies..." -ForegroundColor Yellow
Push-Location webapp
npm ci
Pop-Location
Write-Host "✅ Dependencies installed" -ForegroundColor Green
Write-Host ""

# Сборка плагина
Write-Host "🔧 Building plugin for Linux AMD64..." -ForegroundColor Yellow
make dist-linux
Write-Host "✅ Build complete" -ForegroundColor Green
Write-Host ""

# Проверка результата
$BUNDLE_NAME = "boards-$VERSION.tar.gz"
$bundlePath = "dist\$BUNDLE_NAME"

if (Test-Path $bundlePath) {
    $fileSize = (Get-Item $bundlePath).Length
    $fileSizeMB = [math]::Round($fileSize / 1MB, 2)
    
    Write-Host "✅ Release bundle created successfully!" -ForegroundColor Green
    Write-Host ""
    Write-Host "📦 Bundle: $bundlePath" -ForegroundColor Cyan
    Write-Host "📊 Size: $fileSizeMB MB" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "🚀 You can now upload this file to your Mattermost server" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "To install on server:" -ForegroundColor White
    Write-Host "  scp $bundlePath user@server:/tmp/" -ForegroundColor Gray
    Write-Host "  ssh user@server" -ForegroundColor Gray
    Write-Host "  cd /opt/mattermost/plugins" -ForegroundColor Gray
    Write-Host "  tar -xzf /tmp/$BUNDLE_NAME" -ForegroundColor Gray
    Write-Host "  systemctl restart mattermost" -ForegroundColor Gray
} else {
    Write-Host "❌ Error: Bundle not found at $bundlePath" -ForegroundColor Red
    exit 1
}

