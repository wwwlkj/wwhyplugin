@echo off
echo ====================================
echo 构建 WWPlugin 示例程序
echo ====================================
echo.

echo 1. 构建主机示例...
cd examples\host
go build -o host.exe main.go
if %ERRORLEVEL% EQU 0 (
    echo ✅ 主机示例构建成功
) else (
    echo ❌ 主机示例构建失败
    goto :error
)

echo.
echo 2. 构建插件示例...
cd ..\sample_plugin
go build -o sample_plugin.exe main.go
if %ERRORLEVEL% EQU 0 (
    echo ✅ 插件示例构建成功
) else (
    echo ❌ 插件示例构建失败
    goto :error
)

cd ..\..

echo.
echo ====================================
echo 构建完成！
echo ====================================
echo.
echo 使用方法：
echo 1. 运行主机: examples\host\host.exe
echo 2. 运行插件信息查询: examples\sample_plugin\sample_plugin.exe --info
echo.
echo 注意：主机会自动尝试加载插件，请确保路径正确
echo ====================================
pause
goto :end

:error
echo.
echo ❌ 构建过程中发生错误！
pause

:end