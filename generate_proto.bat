@echo off
echo 正在生成protobuf文件...

REM 设置环境变量
set GOBIN=%GOPATH%\bin
set PATH=%PATH%;%GOBIN%

REM 生成protobuf文件
protoc --go_out=. --go_grpc_out=. proto/plugin.proto

if %ERRORLEVEL% neq 0 (
    echo 生成失败，尝试直接使用工具...
    "%GOPATH%\bin\protoc-gen-go.exe" --version
    "%GOPATH%\bin\protoc-gen-go-grpc.exe" --version
    pause
    exit /b 1
)

echo protobuf文件生成成功！
pause