@echo off
REM GoPay 服务启动脚本 (Windows)
set "COMPOSE_ENV_FILE=%COMPOSE_ENV_FILE%"
if "%COMPOSE_ENV_FILE%"=="" set "COMPOSE_ENV_FILE=.env"

echo ==========================================
echo   启动 GoPay 支付网关服务
echo ==========================================
echo.

REM 检查环境变量文件
if not exist "%COMPOSE_ENV_FILE%" (
    echo ❌ 错误: %COMPOSE_ENV_FILE% 文件不存在
    echo 请复制 .env.example 并配置环境变量
    exit /b 1
)

REM 检查数据库是否启动
echo 📦 检查数据库连接...
podman ps --format "{{.Names}}" | findstr postgres >nul 2>&1
if errorlevel 1 (
    echo ⚠️  数据库未启动，正在启动...
    podman compose --env-file "%COMPOSE_ENV_FILE%" up -d postgres adminer
    echo ⏳ 等待数据库启动...
    timeout /t 5 /nobreak >nul
)

REM 编译服务
echo.
echo 🔨 编译 GoPay 服务...
go build -o bin\gopay.exe cmd\gopay\main.go

if errorlevel 1 (
    echo ❌ 编译失败
    exit /b 1
)

echo ✅ 编译成功
echo.

REM 启动服务
echo 🚀 启动 GoPay 服务...
echo ==========================================
echo.

bin\gopay.exe
