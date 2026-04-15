@echo off
REM GoPay 服务启动脚本 (Windows)

echo ==========================================
echo   启动 GoPay 支付网关服务
echo ==========================================
echo.

REM 检查环境变量文件
if not exist .env (
    echo ❌ 错误: .env 文件不存在
    echo 请复制 .env.example 并配置环境变量
    exit /b 1
)

REM 检查数据库是否启动
echo 📦 检查数据库连接...
docker ps | findstr gopay-postgres >nul 2>&1
if errorlevel 1 (
    echo ⚠️  数据库未启动，正在启动...
    docker-compose up -d
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
