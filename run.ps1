# Запуск API из корня репозитория (подходит для Windows: Go в PATH и корректный DATABASE_URL).
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $MyInvocation.MyCommand.Path
if (-not $root) { $root = Get-Location }
Set-Location $root

# Локальные секреты (в .gitignore): например $env:GEMINI_API_KEY = "..."
$envLocal = Join-Path $root ".env.local.ps1"
if (Test-Path $envLocal) {
    . $envLocal
}

$goExe = Join-Path ${env:ProgramFiles} "Go\bin\go.exe"
if (Test-Path $goExe) {
    $env:Path = "$(Split-Path $goExe);$env:Path"
}

# Переопределяем типичный системный DATABASE_URL на старый localhost:5432 — см. docker-compose (порт 5433, 127.0.0.1).
$env:DATABASE_URL = "postgres://hacknu:hacknu@127.0.0.1:5433/locomotive?sslmode=disable"
if (-not $env:HTTP_ADDR) { $env:HTTP_ADDR = ":8080" }
# Не тащить из сессии IDE отладочный флаг (consumer RabbitMQ включён по умолчанию).
Remove-Item Env:RABBITMQ_DISABLE -ErrorAction SilentlyContinue

& go run ./backend/cmd/server
