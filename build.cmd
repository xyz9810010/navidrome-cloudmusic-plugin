@echo off
rem cloudmusic.ndp 一键构建(官方 Go 工具链,无需 TinyGo)
rem 产物: plugin/cloudmusic.ndp

cd /d %~dp0plugin

echo [1/2] 编译 WASM (wasip1 reactor)...
set GOOS=wasip1
set GOARCH=wasm
go build -buildmode=c-shared -o plugin.wasm .
if errorlevel 1 (
    echo 编译失败
    exit /b 1
)

echo [2/2] 打包 .ndp ...
python -m zipfile -c cloudmusic.ndp plugin.wasm manifest.json
if errorlevel 1 (
    echo 打包失败
    exit /b 1
)

echo 完成: plugin\cloudmusic.ndp
echo 部署: python deploy.py
